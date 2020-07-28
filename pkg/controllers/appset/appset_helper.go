package appset

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/common"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// ApplyStatus modify status handler
func (r *AppSetReconciler) ApplyStatus(ctx context.Context, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) (status workloadv1beta1.AppStatus, isChange bool, err error) {
	as, err := buildAppSetStatus(ctx, r.DksMgr.K8sMgr, req, app)
	if err != nil {
		klog.Errorf("%s: aggregate AppSet.Status failed: %+v", req.NamespacedName, err)
		return "", false, err
	}

	isChange, err = r.applyStatus(ctx, req, app, as)
	return as.AggrStatus.Status, isChange, err
}

func buildAppSetStatus(ctx context.Context, dksManger *k8smanager.ClusterManager, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) (*workloadv1beta1.AppSetStatus, error) {
	changeObserved := true
	finalStatus := workloadv1beta1.AppStatusRuning

	as := &workloadv1beta1.AppSetStatus{
		AggrStatus: workloadv1beta1.AggrAppSetStatus{
			Pods:       []*workloadv1beta1.Pod{},
			Clusters:   []*workloadv1beta1.ClusterAppActual{},
			WarnEvents: []*workloadv1beta1.Event{},
		},
	}

	nameAdvs, err := GetAllClustersAdvDeploymentByApp(dksManger, req.NamespacedName, app)
	if err != nil {
		klog.Warningf("all cluster AdvDeployment get failed, err: %+v", err)
		return nil, err
	}

	warnEvents, err := GetAllClustersEventByApp(dksManger, req.NamespacedName, app)
	if err != nil {
		klog.Warningf("all cluster warn events get failed, err: %+v", err)
		return nil, err
	}

	for _, nameAdv := range nameAdvs {
		adv := nameAdv.Adv
		as.AggrStatus.Version = mergeVersion(as.AggrStatus.Version, adv.Status.AggrStatus.Version)
		as.AggrStatus.Clusters = append(as.AggrStatus.Clusters, &workloadv1beta1.ClusterAppActual{
			Name:        nameAdv.ClusterName,
			Desired:     adv.Status.AggrStatus.Desired,
			Available:   adv.Status.AggrStatus.Available,
			UnAvailable: adv.Status.AggrStatus.UnAvailable,
			PodSets:     adv.Status.AggrStatus.PodSets,
		})

		as.AggrStatus.Available += adv.Status.AggrStatus.Available
		as.AggrStatus.UnAvailable += adv.Status.AggrStatus.UnAvailable
		as.AggrStatus.Desired += adv.Status.AggrStatus.Desired

		if changeObserved {
			changeObserved = adv.ObjectMeta.Generation == adv.Status.ObservedGeneration
		}

		if adv.ObjectMeta.Generation != adv.Status.ObservedGeneration || adv.Status.AggrStatus.Status != workloadv1beta1.AppStatusRuning {
			klog.V(4).Infof("adv name[%s] status is %s, meta generation:%d, observedGeneration:%d",
				req.NamespacedName.Name, adv.Status.AggrStatus.Status, adv.ObjectMeta.Generation, adv.Status.ObservedGeneration)
			finalStatus = workloadv1beta1.AppStatusInstalling
		}
	}

	var replicas int32
	if app.Spec.Replicas != nil {
		replicas = *app.Spec.Replicas
		as.AggrStatus.Desired = *app.Spec.Replicas
	} else {
		replicas = as.AggrStatus.Desired
	}

	// final status aggregate
	if finalStatus == workloadv1beta1.AppStatusRuning && as.AggrStatus.Available == replicas && as.AggrStatus.UnAvailable == 0 {
		as.AggrStatus.Status = workloadv1beta1.AppStatusRuning
	} else {
		as.AggrStatus.Status = workloadv1beta1.AppStatusInstalling
		as.AggrStatus.WarnEvents = warnEvents
	}

	klog.V(5).Infof("%s build status:%s, desired:%d, available:%d, replicas:%d, finalStatus:%s",
		req.NamespacedName, finalStatus, as.AggrStatus.Desired, as.AggrStatus.Available, *app.Spec.Replicas, as.AggrStatus.Status)
	if changeObserved {
		as.ObservedGeneration = app.ObjectMeta.Generation
	} else {
		//minus 1 to case previous fetch cached adv which may cause dirty data in this ObservedGeneration
		as.ObservedGeneration = app.ObjectMeta.Generation - 1
	}

	return as, nil
}

func (r *AppSetReconciler) applyStatus(ctx context.Context, req customctrl.CustomRequest, app *workloadv1beta1.AppSet, as *workloadv1beta1.AppSetStatus) (isChange bool, err error) {
	var change bool
	if app.Status.ObservedGeneration != as.ObservedGeneration {
		app.Status.ObservedGeneration = as.ObservedGeneration
		change = true
	}

	if !change && equality.Semantic.DeepEqual(app.Status.AggrStatus, as.AggrStatus) {
		klog.V(4).Infof("%s/%s status unchanged", req.NamespacedName.Namespace, req.NamespacedName.Name)
		return false, nil
	}

	if as.AggrStatus.Status == workloadv1beta1.AppStatusRuning && app.Status.AggrStatus.Status != workloadv1beta1.AppStatusRuning {
		r.recorder.Event(app, corev1.EventTypeNormal, "Running", "Status is Running.")
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		as.AggrStatus.DeepCopyInto(&app.Status.AggrStatus)
		t := metav1.Now()
		app.Status.LastUpdateTime = &t

		updateErr := r.Client.Status().Update(ctx, app)
		if updateErr == nil {
			klog.V(4).Infof("%s/%s update status[%s] successfully",
				req.NamespacedName.Namespace, req.NamespacedName.Name, app.Status.AggrStatus.Status)
			return nil
		}

		getErr := r.Client.Get(ctx, req.NamespacedName, app)
		if getErr != nil {
			klog.Errorf("%s/%s update get AppSet failed, err: %+v", req.NamespacedName.Namespace, req.NamespacedName.Name, getErr)
			return getErr
		}

		return updateErr
	})
	return true, err
}

// DeleteAll delete crd handler
func (r *AppSetReconciler) DeleteAll(ctx context.Context, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) error {
	// loop cluster delete advdeployment
	for _, cluster := range r.DksMgr.K8sMgr.GetAll() {
		cluster, err := r.DksMgr.K8sMgr.Get(cluster.GetName())
		if err != nil {
			return err
		}

		isChanged, err := deleteByCluster(ctx, cluster, req)
		if err != nil || isChanged {
			return err
		}
	}

	if len(app.ObjectMeta.Finalizers) == 0 {
		klog.V(4).Infof("%s: finalizers is empty", req.NamespacedName)
		return nil
	}

	klog.V(4).Infof("%s: delete all AdvDeployment success, delete AppSet now", req.NamespacedName)
	app.ObjectMeta.Finalizers = utils.RemoveString(app.ObjectMeta.Finalizers, labels.ControllerFinalizersName)
	return r.Client.Update(ctx, app)
}

func deleteByCluster(ctx context.Context, cluster *k8smanager.Cluster, req customctrl.CustomRequest) (bool, error) {
	err := cluster.Client.Delete(ctx, &workloadv1beta1.AdvDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
	})
	if err == nil {
		klog.V(4).Infof("%s: delete cluster[%s] Advdeployment success", req.NamespacedName, cluster.GetName())
		return true, nil
	}
	if apierrors.IsNotFound(err) {
		klog.Errorf("%s: delete cluster[%s] Advdeployment fail, not found", req.NamespacedName, cluster.GetName())
		return false, nil
	}

	return false, fmt.Errorf("delete cluster[%s] AdvDeployment(%s) fail:%+v", cluster.GetName(), req.NamespacedName.String(), err)
}

func (r *AppSetReconciler) DeleteUnuseAdvDeployment(ctx context.Context, req customctrl.CustomRequest, status workloadv1beta1.AppStatus) (isChange bool, err error) {
	if status != workloadv1beta1.AppStatusRuning {
		return false, nil
	}

	app := &workloadv1beta1.AppSet{}
	if err := r.Client.Get(ctx, req.NamespacedName, app); err != nil {
		klog.Errorf("%s: get AppSet info faild, err: %+v", req.NamespacedName, err)
		return false, err
	}

	// get current info
	currentInfo := map[string]*workloadv1beta1.AdvDeployment{}
	for _, cluster := range r.DksMgr.K8sMgr.GetAll() {
		b := &workloadv1beta1.AdvDeployment{}
		err := cluster.Client.Get(ctx, req.NamespacedName, b)
		if err == nil {
			currentInfo[cluster.GetName()] = b
			continue
		}
		if apierrors.IsNotFound(err) {
			continue
		}
		return false, err
	}

	// build expect info with app
	expectInfo := map[string]struct{}{}
	for _, cluster := range app.Spec.ClusterTopology.Clusters {
		expectInfo[cluster.Name] = struct{}{}
	}

	// current equal expect
	if len(expectInfo) == len(currentInfo) {
		return false, nil
	}

	delCluster := ""
	for current := range currentInfo {
		if _, ok := expectInfo[current]; ok {
			continue
		}
		delCluster = current
		break
	}
	if delCluster == "" {
		// not exist need delete cluster
		return false, nil
	}
	klog.V(4).Infof("%s: delete unexpect info cluster:%s", req.NamespacedName, delCluster)

	client, err := r.DksMgr.K8sMgr.Get(delCluster)
	if err != nil {
		klog.Errorf("%s: delete unexpect info, get cluster[%s] client err: %+v", req.NamespacedName, delCluster, err)
		return false, err
	}
	return deleteByCluster(ctx, client, req)
}

// ApplySpec
func (r *AppSetReconciler) ApplySpec(ctx context.Context, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) (int, error) {
	var changed int

	for _, v := range app.Spec.ClusterTopology.Clusters {
		c, err := r.DksMgr.K8sMgr.Get(v.Name)
		if err != nil {
			return 0, errors.Wrapf(err, "cluster: %s is offline", v.Name)
		}

		newObjAdv := buildAdvDeployment(app, v, r.DksMgr.Opt.Debug)
		isChanged, err := resources.Reconcile(ctx, c.Client, newObjAdv, resources.Option{})
		if err != nil {
			return 0, err
		}

		if isChanged > 0 {
			changed++
		}
		klog.V(5).Infof("[%s/%s] apply AdvDeployment by cluster[%s] successfully",
			req.NamespacedName.Namespace, req.NamespacedName.Name, v.Name)
	}

	return changed, nil
}

func buildAdvDeployment(app *workloadv1beta1.AppSet, clusterTopology *workloadv1beta1.TargetCluster, debug bool) *workloadv1beta1.AdvDeployment {
	replica := 0
	for _, v := range clusterTopology.PodSets {
		replica += v.Replicas.IntValue()
	}

	obj := &workloadv1beta1.AdvDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        app.Name,
			Namespace:   app.Namespace,
			Labels:      makeAdvDeploymentLabel(clusterTopology, app),
			Annotations: makeAdvDeploymentAnnotation(app),
		},
	}

	if app.Spec.ServiceName != nil {
		svcName := common.FormatToDNS1123(*app.Spec.ServiceName)
		obj.Spec.ServiceName = &svcName
	}

	obj.Spec.Replicas = utils.IntPointer(int32(replica))
	app.Spec.PodSpec.DeepCopyInto(&obj.Spec.PodSpec)

	for _, set := range clusterTopology.PodSets {
		podSet := set.DeepCopy()
		if len(podSet.RawValues) == 0 && debug {
			podSet.RawValues = makeHelmOverrideValus(podSet.Name, clusterTopology, app)
		}
		obj.Spec.Topology.PodSets = append(obj.Spec.Topology.PodSets, podSet)
	}
	return obj
}
