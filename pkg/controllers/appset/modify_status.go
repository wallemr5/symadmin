package appset

import (
	"context"
	"fmt"
	"sort"
	"strings"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
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

	as, err := buildAppSetStatus(ctx, r.DksMgr.K8sMgr, req, app)
	if err != nil {
		klog.Errorf("%s: aggregate AppSet.Status failed: %+v", req.NamespacedName, err)
		return "", false, err
	}
	if as.Status == workloadv1beta1.AppStatusRuning {
		r.recorder.Event(app, corev1.EventTypeNormal, "Running", "Status is Running.")
	}

	isChange, err = applyStatus(ctx, r.Client, req, app, as)
	return as.Status, isChange, err
}

func buildAppSetStatus(ctx context.Context, dksManger *k8smanager.ClusterManager, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) (*workloadv1beta1.AggrAppSetStatus, error) {
	as := &workloadv1beta1.AggrAppSetStatus{
		Clusters: []*workloadv1beta1.ClusterAppActual{},
	}
	finalStatus := workloadv1beta1.AppStatusRuning

	for _, cluster := range app.Spec.ClusterTopology.Clusters {
		cli, err := dksManger.Get(cluster.Name)
		if err != nil {
			return nil, err
		}

		obj := &workloadv1beta1.AdvDeployment{}
		if err := cli.Cache.Get(ctx, req.NamespacedName, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("get cluster:%s AdvDeployment info err:%+v", cluster.Name, err)
		}

		// aggregate status
		as.Available += obj.Status.AggrStatus.Available
		as.UnAvailable += obj.Status.AggrStatus.UnAvailable
		as.Desired += obj.Status.AggrStatus.Desired
		if obj.ObjectMeta.Generation != obj.Status.ObservedGeneration || obj.Status.AggrStatus.Status != workloadv1beta1.AppStatusRuning {
			klog.V(4).Infof("%s: cluster[%s] status is %s, meta generation:%d, observedGeneration:%d", req.NamespacedName, cluster.Name, obj.Status.AggrStatus.Status, obj.ObjectMeta.Generation, obj.Status.ObservedGeneration)
			finalStatus = workloadv1beta1.AppStatusInstalling
		}
		if !strings.Contains(as.Version, obj.Status.AggrStatus.Version) {
			as.Version = mergeVersion(as.Version, obj.Status.AggrStatus.Version)
		}
		as.Clusters = append(as.Clusters, &workloadv1beta1.ClusterAppActual{
			Name:        cluster.Name,
			Desired:     obj.Status.AggrStatus.Desired,
			Available:   obj.Status.AggrStatus.Available,
			UnAvailable: obj.Status.AggrStatus.UnAvailable,
			PodSets:     obj.Status.AggrStatus.PodSets,
		})

		events := &corev1.EventList{}
		opt := &client.ListOptions{}
		if err := cli.Cache.List(ctx, opt.InNamespace(req.Namespace), events); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("get cluster[%s] events list err: %+v", cluster.Name, err)
		}

		// aggregate events
		evts := []*workloadv1beta1.Event{}
		for _, evt := range events.Items {
			if strings.HasPrefix(evt.InvolvedObject.Name, req.Name) && evt.Type == corev1.EventTypeWarning {
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

		if as.WarnEvents == nil {
			as.WarnEvents = []*workloadv1beta1.Event{}
		}
		as.WarnEvents = evts
	}

	// final status aggregate
	if finalStatus == workloadv1beta1.AppStatusRuning && as.Available == *app.Spec.Replicas {
		as.Status = workloadv1beta1.AppStatusRuning
	} else {
		as.Status = workloadv1beta1.AppStatusInstalling
	}
	if app.Spec.Replicas != nil {
		as.Desired = *app.Spec.Replicas
	}
	klog.V(4).Infof("%s: build AppSet.Status.Aggregate.Status judgeStatus:%s available:%d replicas:%d, finalStatus:%s", req.NamespacedName, finalStatus, as.Available, *app.Spec.Replicas, as.Status)

	return as, nil
}

func applyStatus(ctx context.Context, client client.Client, req customctrl.CustomRequest, app *workloadv1beta1.AppSet, as *workloadv1beta1.AggrAppSetStatus) (isChange bool, err error) {

	if equality.Semantic.DeepEqual(&app.Status.AggrStatus, as) {
		klog.V(4).Infof("%s: applyStatus AppSet.Status not need change", req.NamespacedName)
		return false, nil
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {

		as.DeepCopyInto(&app.Status.AggrStatus)
		t := metav1.Now()
		app.Status.LastUpdateTime = &t

		updateErr := client.Status().Update(ctx, app)
		if updateErr == nil {
			klog.V(4).Infof("%s: applyStatus update AppSet.Status.AggrStatus.Status success: %s", req.NamespacedName, app.Status.AggrStatus.Status)
			return nil
		}

		getErr := client.Get(ctx, req.NamespacedName, app)
		if getErr != nil {
			klog.Errorf("%s: applyStatus get AppSet again info fail: %+v", req.NamespacedName, getErr)
			return getErr
		}

		return updateErr
	})
	return true, err
}
