package appset

import (
	"context"
	"fmt"
	"sort"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ModifyStatus modify status handler
func (r *AppSetReconciler) ModifyStatus(ctx context.Context, req customctrl.CustomRequest) (status workloadv1beta1.AppStatus, isChange bool, err error) {
	app := &workloadv1beta1.AppSet{}
	if err := r.Client.Get(ctx, req.NamespacedName, app); err != nil {
		klog.Errorf("%s: applyStatus get AppSet info fail: %+v", req.NamespacedName, err)
		return "", false, err
	}

	as, err := buildAppSetStatus2(ctx, r.DksMgr.K8sMgr, req, app)
	if err != nil {
		klog.Errorf("%s: aggregate AppSet.Status failed: %+v", req.NamespacedName, err)
		return "", false, err
	}

	isChange, err = r.applyStatus(ctx, req, app, as)
	return as.AggrStatus.Status, isChange, err
}

func buildAppSetStatus(ctx context.Context, dksManger *k8smanager.ClusterManager, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) (*workloadv1beta1.AppSetStatus, error) {
	as := &workloadv1beta1.AppSetStatus{
		AggrStatus: workloadv1beta1.AggrAppSetStatus{
			Pods:       []*workloadv1beta1.Pod{},
			Clusters:   []*workloadv1beta1.ClusterAppActual{},
			WarnEvents: []*workloadv1beta1.Event{},
		},
	}

	finalStatus := workloadv1beta1.AppStatusRuning
	changeObserved := true

	for _, cluster := range app.Spec.ClusterTopology.Clusters {
		cli, err := dksManger.Get(cluster.Name)
		if err != nil {
			return nil, err
		}

		obj := &workloadv1beta1.AdvDeployment{}
		if err := cli.Client.Get(ctx, req.NamespacedName, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("get cluster:%s AdvDeployment info err:%+v", cluster.Name, err)
		}

		// aggregate status
		as.AggrStatus.Available += obj.Status.AggrStatus.Available
		as.AggrStatus.UnAvailable += obj.Status.AggrStatus.UnAvailable
		as.AggrStatus.Desired += obj.Status.AggrStatus.Desired

		if changeObserved {
			changeObserved = obj.ObjectMeta.Generation == obj.Status.ObservedGeneration
		}

		if obj.ObjectMeta.Generation != obj.Status.ObservedGeneration || obj.Status.AggrStatus.Status != workloadv1beta1.AppStatusRuning {
			klog.V(4).Infof("%s: cluster[%s] status is %s, meta generation:%d, observedGeneration:%d", req.NamespacedName, cluster.Name, obj.Status.AggrStatus.Status, obj.ObjectMeta.Generation, obj.Status.ObservedGeneration)
			finalStatus = workloadv1beta1.AppStatusInstalling
		}
		as.AggrStatus.Version = mergeVersion(as.AggrStatus.Version, obj.Status.AggrStatus.Version)
		as.AggrStatus.Clusters = append(as.AggrStatus.Clusters, &workloadv1beta1.ClusterAppActual{
			Name:        cluster.Name,
			Desired:     obj.Status.AggrStatus.Desired,
			Available:   obj.Status.AggrStatus.Available,
			UnAvailable: obj.Status.AggrStatus.UnAvailable,
			PodSets:     obj.Status.AggrStatus.PodSets,
		})

		events := &corev1.EventList{}
		opt := &client.ListOptions{}
		if err := cli.Client.List(ctx, opt.InNamespace(req.Namespace), events); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("get cluster[%s] events list err: %+v", cluster.Name, err)
		}

		// aggregate events
		evts := []*workloadv1beta1.Event{}
		for i := range events.Items {
			evt := &events.Items[i]
			if isAppendEvt(evt, app) {
				evts = append(evts, &workloadv1beta1.Event{
					Message:         evt.Message,
					SourceComponent: evt.Source.Component,
					Name:            evt.Name,
					Count:           evt.Count,
					FirstSeen:       evt.FirstTimestamp,
					LastSeen:        evt.LastTimestamp,
					Reason:          evt.Reason,
					Type:            evt.Type,
				})
			}
		}
		sort.Slice(evts, func(i int, j int) bool {
			return evts[i].Name > evts[j].Name && evts[i].Reason > evts[j].Reason
		})

		as.AggrStatus.WarnEvents = evts
	}

	// final status aggregate
	if finalStatus == workloadv1beta1.AppStatusRuning &&
		as.AggrStatus.Available == *app.Spec.Replicas &&
		as.AggrStatus.Available == as.AggrStatus.Desired &&
		as.AggrStatus.UnAvailable == 0 {
		as.AggrStatus.Status = workloadv1beta1.AppStatusRuning
		as.AggrStatus.WarnEvents = nil
	} else {
		as.AggrStatus.Status = workloadv1beta1.AppStatusInstalling
	}
	if app.Spec.Replicas != nil {
		as.AggrStatus.Desired = *app.Spec.Replicas
	}
	klog.V(4).Infof("%s: build status judgeStatus:%s, desired:%d, available:%d, replicas:%d, finalStatus:%s", req.NamespacedName, finalStatus, as.AggrStatus.Desired, as.AggrStatus.Available, *app.Spec.Replicas, as.AggrStatus.Status)

	if changeObserved {
		as.ObservedGeneration = app.ObjectMeta.Generation
	} else {
		as.ObservedGeneration = app.Status.ObservedGeneration
	}

	return as, nil
}

func buildAppSetStatus2(ctx context.Context, dksManger *k8smanager.ClusterManager, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) (*workloadv1beta1.AppSetStatus, error) {
	as := &workloadv1beta1.AppSetStatus{
		AggrStatus: workloadv1beta1.AggrAppSetStatus{
			Pods:       []*workloadv1beta1.Pod{},
			Clusters:   []*workloadv1beta1.ClusterAppActual{},
			WarnEvents: []*workloadv1beta1.Event{},
		},
	}

	finalStatus := workloadv1beta1.AppStatusRuning
	changeObserved := true

	nameAdvs, err := GetAllClustersAdvDeploymentByApp(dksManger, req.NamespacedName, app)
	if err != nil {
		klog.Warningf("all cluster AdvDeployment get fail, err: %+v", err)
		return nil, err
	}

	warnEvents, err := GetAllClustersEventByApp(dksManger, req.NamespacedName, app)
	if err != nil {
		klog.Warningf("all cluster warn events get fail, err: %+v", err)
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
			klog.Errorf("%s/%s update get AppSet fail, err: %+v", req.NamespacedName.Namespace, req.NamespacedName.Name, getErr)
			return getErr
		}

		return updateErr
	})
	return true, err
}

func isAppendEvt(evt *corev1.Event, app *workloadv1beta1.AppSet) bool {
	if evt == nil || app == nil {
		return false
	}

	if evt.Type == corev1.EventTypeNormal {
		return false
	}

	if app.ObjectMeta.CreationTimestamp.After(evt.LastTimestamp.Time) {
		return false
	}

	if labels.CheckEventLabel(evt.InvolvedObject.Name, app.Name) {
		return true
	}

	if evt.InvolvedObject.Kind == "AdvDeployment" && evt.InvolvedObject.Name == app.Name {
		return true
	}
	return false
}
