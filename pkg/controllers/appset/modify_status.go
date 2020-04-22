package appset

import (
	"context"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// ModifyStatus modify status handler
func (r *AppSetReconciler) ModifyStatus(ctx context.Context, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) (status workloadv1beta1.AppStatus, isChange bool, err error) {
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
		as.ObservedGeneration = app.Status.ObservedGeneration
	}

	return as, nil
}

func (r *AppSetReconciler) applyStatus(ctx context.Context, req customctrl.CustomRequest, app *workloadv1beta1.AppSet, as *workloadv1beta1.AppSetStatus) (isChange bool, err error) {
	var change bool
	if app.Status.ObservedGeneration != as.ObservedGeneration {
		app.Status.ObservedGeneration = app.ObjectMeta.Generation
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
