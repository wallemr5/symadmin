package cluster

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	sym_api "gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/api"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/common"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/contour"
	sym_ctl "gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/controller"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/monitor"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/other"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/traefik"
	helmv3 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v3"
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
	Log        logr.Logger
	Mgr        manager.Manager
	KubeCli    kubernetes.Interface
	Cfg        *rest.Config
	DksMgr     *pkgmanager.DksManager
	HelmSyncer *helmv3.HelmIndexSyncer
	Clusters   map[string]*k8smanager.Cluster
}

// Add add controller to runtime manager
func Add(mgr manager.Manager, cMgr *pkgmanager.DksManager) error {
	r := &Reconciler{
		Name:     "cluster-controllers",
		Client:   mgr.GetClient(),
		Mgr:      mgr,
		DksMgr:   cMgr,
		Log:      ctrl.Log.WithName("cluster"),
		Clusters: make(map[string]*k8smanager.Cluster),
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

	helmvEnv, err := helmv3.InitHelmRepoEnv("dmall", cMgr.Opt.Repos)
	if err != nil {
		klog.Errorf("Initializing a helm env has an error:%v", err)
	}
	r.HelmSyncer = helmv3.NewDefaultHelmIndexSyncer(helmvEnv)

	klog.Infof("add helm repo index syncer Runnable")
	mgr.Add(r.HelmSyncer)
	return nil
}

// +kubebuilder:rbac:groups=workload.dmall.com,resources=advdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workload.dmall.com,resources=advdeployments/status,verbs=get;update;patch

// Reconcile ...
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
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
			logger.Info("not find cluster")
			return reconcile.Result{}, nil
		}

		logger.Error(err, "failed to get cluster")
		return reconcile.Result{}, err
	}

	if cluster.Spec.Pause {
		return reconcile.Result{}, nil
	}

	isNeedUpdate, err := r.reconcile(ctx, cluster)
	if err != nil {
		logger.Error(err, "after reconcile")
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
func (r *Reconciler) EnsureClusters(namespace string, clusterName string) (*k8smanager.Cluster, error) {
	var k *k8smanager.Cluster
	if clusterName == "" {
		return nil, errors.New("clusterName is empty")
	}

	// find global manager cluster
	k, err := r.DksMgr.K8sMgr.Get(clusterName)
	if err == nil && k != nil {
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

	nc, err := k8smanager.NewCluster(clusterName, kubeconfig, r.Log)
	if err != nil {
		klog.Errorf("cluster: %s new client err: %v", clusterName, err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	nc.StartCache(ctx.Done())
	r.Clusters[clusterName] = nc

	klog.Infof("start custom cluster[%s] cache success", clusterName)
	return nc, nil
}

func (r *Reconciler) reconcile(ctx context.Context, obj *workloadv1beta1.Cluster) (int, error) {
	var isNeedUpdate int
	var err error

	k, err := r.EnsureClusters(obj.Namespace, obj.Name)
	if err != nil {
		return isNeedUpdate, errors.Wrapf(err, "clusterName: %s EnsureClustes", obj.Name)
	}

	if obj.Status.Version == nil {
		info, err := k.KubeCli.Discovery().ServerVersion()
		if err != nil {
			return isNeedUpdate, errors.Wrapf(err, "name: %s get version ", obj.Name)
		}
		obj.Status.Version = info
		isNeedUpdate++
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

	newobj.Annotations = obj.Annotations
	_ = r.Client.Update(ctx, newobj)
	return newobj, err
}

func (r *Reconciler) reconcileComponent(ctx context.Context, kcli *k8smanager.Cluster, org *workloadv1beta1.Cluster) (int, error) {
	if org.Spec.SymNodeName == "" {
		klog.Infof("====> cluster:%s no SymNodeName", org.Name)
		return 0, nil
	}

	obj := org.DeepCopy()
	err := common.PreLabelsNs(kcli, obj)
	if err != nil {
		return 0, err
	}

	err = common.PreLabelsNode(kcli, obj)
	if err != nil {
		return 0, err
	}

	phases := []common.ComponentReconciler{
		other.New(kcli, obj, r.HelmSyncer.HelmEnv),
		traefik.New(kcli, obj, r.HelmSyncer.HelmEnv),
		contour.New(kcli, obj, r.HelmSyncer.HelmEnv),
		monitor.New(r.Mgr, kcli, obj, r.HelmSyncer.HelmEnv),
		sym_ctl.New(r.Mgr, kcli, obj, r.HelmSyncer.HelmEnv),
		sym_api.New(kcli, obj, r.HelmSyncer.HelmEnv),
	}

	appHelms := make([]*workloadv1beta1.AppHelmStatus, 0, len(obj.Spec.Apps))
	for _, app := range obj.Spec.Apps {
		phase, err := common.FindComponentReconciler(app.Name, phases)
		if err != nil {
			klog.Errorf("name[%s] err: %v", app.Name, err)
			continue
		}

		info, rerr := phase.Reconcile(r.Log, app)
		if rerr != nil {
			klog.Errorf("app: %s Reconcile err: %#v", app.Name, rerr)
			appHelms = append(appHelms, &workloadv1beta1.AppHelmStatus{
				Name:         app.Name,
				ChartVersion: app.ChartVersion,
				RlsStatus:    rerr.Error(),
			})
			continue
		}

		if st, ok := info.(*workloadv1beta1.AppHelmStatus); ok {
			appHelms = append(appHelms, st)
		}
	}
	sort.Slice(appHelms, func(i, j int) bool {
		return appHelms[i].RlsName < appHelms[j].RlsName
	})

	var isChanged int
	isSame := equality.Semantic.DeepEqual(appHelms, obj.Status.AppHelms)
	if !isSame {
		org.Status.AppHelms = appHelms
		isChanged++
	} else {
		klog.V(3).Infof("clusterName:%s helm appStatus is same, ignore", kcli.Name)
	}

	if !equality.Semantic.DeepEqual(org.Annotations, obj.Annotations) {
		org.Annotations = obj.Annotations
		isChanged++
	}

	return isChanged, nil
}
