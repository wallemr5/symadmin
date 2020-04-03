package cluster

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/common"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/monitor"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/other"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/swift"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/traefik"
	clusterutils "gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/utils"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	"gitlab.dmall.com/arch/sym-admin/pkg/helm/v2repo"
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "cluster-controller"
)

// Reconciler reconciles a cluster.workload.dmall.com object
type Reconciler struct {
	Name string
	client.Client
	Log      logr.Logger
	Mgr      manager.Manager
	KubeCli  kubernetes.Interface
	Cfg      *rest.Config
	DksMgr   *pkgmanager.DksManager
	HelmEnv  *v2repo.HelmIndexSyncer
	Clusters map[string]*k8smanager.Cluster
}

// Add add controller to runtime manager
func Add(mgr manager.Manager, cMgr *pkgmanager.DksManager) error {
	r := &Reconciler{
		Name:   "cluster-controllers",
		Client: mgr.GetClient(),
		Mgr:    mgr,
		DksMgr: cMgr,
		Log:    ctrl.Log.WithName("controllers").WithName("cluster"),
	}

	r.Cfg = mgr.GetConfig()
	kubeCli, err := k8sclient.NewClientFromConfig(mgr.GetConfig())
	if err != nil {
		r.Log.Error(err, "Creating a kube client for the reconciler has an error")
		return err
	}
	r.KubeCli = kubeCli

	// Create a new runtime controller for advDeployment
	ctl, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r, MaxConcurrentReconciles: cMgr.Opt.Threadiness})
	if err != nil {
		r.Log.Error(err, "Creating a new cluster controller has an error")
		return err
	}

	// We set the objects which would to be watched by this controller.
	err = ctl.Watch(&source.Kind{Type: &workloadv1beta1.Cluster{}}, &handler.EnqueueRequestForObject{}, utils.GetWatchPredicateForClusterSpec())
	if err != nil {
		r.Log.Error(err, "Watching cluster has an error")
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&workloadv1beta1.Cluster{}).
		WithEventFilter(utils.GetWatchPredicateForClusterSpec()).
		Complete(r)
	if err != nil {
		r.Log.Error(err, "could not create controller")
		return err
	}

	helmv2env, err := helmv2.InitHelmRepoEnv("dmall", cMgr.Opt.Repos)
	if err != nil {
		klog.Errorf("Initializing a helm env has an error:%v", err)
	}
	r.HelmEnv = v2repo.NewDefaultHelmIndexSyncer(helmv2env)

	klog.Infof("add helm repo index syncer Runnable")
	mgr.Add(r.HelmEnv)
	return nil
}

// +kubebuilder:rbac:groups=workload.dmall.com,resources=advdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workload.dmall.com,resources=advdeployments/status,verbs=get;update;patch

// Reconcile ...
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	klog.V(3).Infof("##### [%s] start to reconcile.", req.NamespacedName)

	// Calculating how long did the reconciling process take
	startTime := time.Now()
	defer func() {
		diffTime := time.Since(startTime)
		var logLevel klog.Level
		if diffTime > 2*time.Second {
			logLevel = 2
		} else if diffTime > 1*time.Second {
			logLevel = 3
		} else {
			logLevel = 4
		}
		klog.V(logLevel).Infof("##### [%s] reconciling is finished. time taken: %v. ", req.NamespacedName, diffTime)
	}()

	ctx := context.Background()
	logger := r.Log.WithValues("key", req.NamespacedName, "id", uuid.Must(uuid.NewV4()).String())

	cluster := &workloadv1beta1.Cluster{}
	err := r.Client.Get(ctx, req.NamespacedName, cluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.V(3).Infof("Can not find cluster name [%s/%s]", req.NamespacedName.Namespace, req.NamespacedName.Name)
			return reconcile.Result{}, nil
		}

		logger.Error(err, "failed to get cluster")
		return reconcile.Result{}, err
	}

	if cluster.Spec.Pause {
		return reconcile.Result{}, nil
	}

	k, err := r.EnsureClustes(cluster.Namespace, cluster.Name)
	if err != nil {
		return ctrl.Result{}, errors.WithMessagef(err, "clusterName: %s", cluster.Name)
	}

	isNeedUpdate, err := r.reconcile(ctx, k, cluster)
	if err != nil {
		klog.Errorf("reconcile err:%+v", err)
	}
	if isNeedUpdate > 0 {
		_, _ = r.UpdateCluster(ctx, cluster)
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) getK8SConfigForMaster(namespace string, name string) ([]byte, error) {
	var configMap corev1.ConfigMap
	err := r.Client.Get(context.TODO(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &configMap)
	if err != nil {
		return nil, err
	}

	for _, config := range configMap.Data {
		return []byte(config), nil
	}

	return nil, fmt.Errorf("could not found kubeconfig name[%s] config from configmap", name)
}

// EnsureClustes ...
func (r *Reconciler) EnsureClustes(namespace string, clusterName string) (*k8smanager.Cluster, error) {
	var k *k8smanager.Cluster
	if clusterName == "" {
		return nil, errors.New("clusterName is empty")
	}

	// find global manager cluster
	k, err := r.DksMgr.K8sMgr.Get(clusterName)
	if err != nil {
		return nil, err
	}
	if k != nil {
		return k, nil
	}

	// find custom manager cluster
	if k, ok := r.Clusters[clusterName]; ok {
		return k, nil
	}

	kubeconfig, err := r.getK8SConfigForMaster(namespace, clusterName)
	if err != nil {
		return nil, err
	}

	nc, err := k8smanager.NewCluster(clusterName, kubeconfig, nil)
	if err != nil {
		klog.Errorf("cluster: %s new client err: %v", clusterName, err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	nc.StartCache(ctx.Done())
	r.Clusters[clusterName] = nc

	klog.Infof("start custom cluster[%s] cache success", clusterName)
	return k, nil
}

func (r *Reconciler) reconcile(ctx context.Context, k *k8smanager.Cluster, obj *workloadv1beta1.Cluster) (int, error) {
	var isNeedUpdate int

	if obj.Status.Version == nil {
		info, err := k.KubeCli.Discovery().ServerVersion()
		if err != nil {
			return isNeedUpdate, errors.Wrapf(err, "name: %s get version ", obj.Name)
		}
		obj.Status.Version = info
		isNeedUpdate++
	}

	isNeedUpdate, err := r.reconcileHelmTiller(ctx, k, obj)
	if err != nil {
		klog.Errorf("cluster:%s reconcileHelmTiller err:%+v", obj.Name, err)
		return isNeedUpdate, err
	}

	isNeedUpdate, err = r.reconcileComponent(ctx, k, obj)
	if err != nil {
		return isNeedUpdate, err
	}

	return isNeedUpdate, nil
}

// UpdateCluster ...
func (r *Reconciler) UpdateCluster(ctx context.Context, obj *workloadv1beta1.Cluster) (*workloadv1beta1.Cluster, error) {
	nsName := types.NamespacedName{
		Name:      obj.Name,
		Namespace: obj.Namespace,
	}

	newobj := obj.DeepCopy()
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		updateErr := r.Client.Status().Update(ctx, newobj)
		if updateErr == nil {
			klog.V(3).Infof("===> Cluster: [%s/%s] updated successfully", newobj.Namespace, newobj.Name)
			return nil
		}

		klog.Errorf("failed to update Cluster: [%s/%s], error: %v", newobj.Namespace, newobj.Name, updateErr)
		getErr := r.Client.Get(ctx, nsName, newobj)
		if getErr != nil {
			utilruntime.HandleError(fmt.Errorf("getting updated Status advDeploy: [%s/%s] err: %v", newobj.Namespace, newobj.Name, getErr))
		}

		obj.Status.DeepCopyInto(&newobj.Status)
		return updateErr
	})

	return newobj, err
}

func (r *Reconciler) reconcileHelmTiller(ctx context.Context, k *k8smanager.Cluster, obj *workloadv1beta1.Cluster) (int, error) {
	var isNeedUpdate int
	var helmSpec workloadv1beta1.HelmSpec
	var isTillerRunning, isNodeAffinity bool

	if obj.Spec.HelmSpec != nil {
		tillerDeploy, err := clusterutils.GetTillerDeploy(ctx, k)
		if err != nil {
			klog.Errorf("Client get tiller err:%+v", err)
			return isNeedUpdate, err
		}
		if tillerDeploy != nil {
			klog.V(3).Infof("deploy name: %s, namespace: %s", tillerDeploy.Name, tillerDeploy.Namespace)
			helmSpec.Namespace = tillerDeploy.Namespace
			for _, container := range tillerDeploy.Spec.Template.Spec.Containers {
				if container.Name == clusterutils.TillerContainerName {
					helmSpec.OverrideImageSpec = container.Image
					for _, env := range container.Env {
						if env.Name == clusterutils.TillerHistoryMax {
							intValue, _ := strconv.Atoi(env.Value)
							helmSpec.MaxHistory = intValue
						}
					}
				}
			}

			isNodeAffinity = tillerDeploy.Spec.Template.Spec.Affinity != nil
			isTillerRunning = tillerDeploy.Status.Replicas > 0 && tillerDeploy.Status.AvailableReplicas == tillerDeploy.Status.Replicas
		}

		if obj.Spec.HelmSpec.Namespace == "" {
			obj.Spec.HelmSpec.Namespace = clusterutils.TillerNameSpace
			isNeedUpdate++
		}

		if helmSpec.OverrideImageSpec == "" {
			err := clusterutils.InstallTiller(k, obj)
			if err != nil {
				return isNeedUpdate, err
			}
		} else {
			if helmSpec.OverrideImageSpec != obj.Spec.HelmSpec.OverrideImageSpec ||
				helmSpec.MaxHistory != obj.Spec.HelmSpec.MaxHistory ||
				isNodeAffinity == false {
				klog.Infof("cluster:%s starting upgrade deploy tiller, spec:%+v", k.Name, obj.Spec.HelmSpec)
				err := clusterutils.UpgradeTiller(k, obj)
				if err != nil {
					return isNeedUpdate, err
				}
			} else {
				klog.V(3).Infof("cluster:%s tiller spec is same, ignore", k.Name)
			}
		}
	}

	if !isTillerRunning {
		return isNeedUpdate, errors.New("tiller deploy pod is not available")
	}
	return isNeedUpdate, nil
}

func (r *Reconciler) reconcileComponent(ctx context.Context, k *k8smanager.Cluster, obj *workloadv1beta1.Cluster) (int, error) {
	if obj.Spec.SymNodeName == "" {
		klog.Infof("====> cluster:%s no SymNodeName", obj.Name)
		return 0, nil
	}
	err := common.PreLabelsNs(k, obj)
	if err != nil {
		return 0, err
	}

	err = common.PreLabelsNode(k, obj)
	if err != nil {
		return 0, err
	}

	hClient, err := helmv2.NewClientFromConfig(k.RestConfig, k.KubeCli, k.Name, r.HelmEnv.Helmv2env)
	if err != nil {
		klog.Errorf("clusterName:%s New hClinet err:%+v", k.Name, err)
		return 0, err
	}
	defer hClient.Close()

	phases := []common.ComponentReconciler{
		other.New(k, obj, hClient),
		traefik.New(k, obj, hClient),
		monitor.New(r.Mgr, k, obj, hClient),
		swift.New(k, obj, hClient),
	}

	appStatus := make([]*workloadv1beta1.AppHelmStatuses, 0, len(obj.Spec.Apps))
	for _, app := range obj.Spec.Apps {
		phase, err := common.FindComponentReconciler(app.Name, phases)
		if err != nil {
			klog.Errorf("name[%s] err: %v", app.Name, err)
			continue
		}

		info, rerr := phase.Reconcile(r.Log, app)
		if rerr != nil {
			klog.Errorf("app: %s Reconcile err: %v", app.Name, rerr)
			continue
		}

		if st, ok := info.(*workloadv1beta1.AppHelmStatuses); ok {
			appStatus = append(appStatus, st)
		}
	}
	sort.Slice(appStatus, func(i, j int) bool {
		return appStatus[i].RlsName < appStatus[j].RlsName
	})

	isSame := equality.Semantic.DeepEqual(appStatus, obj.Status.AppStatus)
	if isSame {
		klog.V(3).Infof("clusterName:%s helm appStatus is same, ignore", k.Name)
		return 0, nil
	}

	obj.Status.AppStatus = appStatus
	return 1, nil
}
