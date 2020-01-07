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
func (r *AppSetReconciler) ModifyStatus(ctx context.Context, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) error {
	as, err := buildAppSetStatus(ctx, r.DksMgr.K8sMgr, req, app)
	if err != nil {
		klog.V(4).Infof("%s: aggregate AppSet.Status failed: %+v", req.NamespacedName, err)
		return err
	}

	if err := applyStatus(ctx, r.Client, req, as); err != nil {
		return err
	}
	return nil
}

func buildAppSetStatus(ctx context.Context, dksManger *k8smanager.ClusterManager, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) (*workloadv1beta1.AggrAppSetStatus, error) {
	as := &workloadv1beta1.AggrAppSetStatus{
		Clusters: []*workloadv1beta1.ClusterAppActual{},
	}
	finalStatus := workloadv1beta1.AppStatusRuning

	for _, cluster := range dksManger.GetAllSort() {
		if cluster.Status == k8smanager.ClusterOffline {
			continue
		}

		obj := &workloadv1beta1.AdvDeployment{}
		if err := cluster.Cache.Get(ctx, req.NamespacedName, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("get cluster:%s AdvDeployment info err:%+v", cluster.GetName(), err)
		}

		// aggregate status
		as.Available += obj.Status.AggrStatus.Desired
		as.UnAvailable += obj.Status.AggrStatus.UnAvailable
		as.Desired += obj.Status.AggrStatus.Desired
		if obj.Status.AggrStatus.Status != workloadv1beta1.AppStatusRuning {
			finalStatus = workloadv1beta1.AppStatusInstalling
		}
		if !strings.Contains(as.Version, obj.Status.AggrStatus.Version) {
			as.Version = mergeVersion(as.Version, obj.Status.AggrStatus.Version)
		}
		as.Clusters = append(as.Clusters, &workloadv1beta1.ClusterAppActual{
			Desired:     obj.Status.AggrStatus.Desired,
			Available:   obj.Status.AggrStatus.Available,
			UnAvailable: obj.Status.AggrStatus.UnAvailable,
			PodSets:     obj.Status.AggrStatus.PodSets,
		})

		events := &corev1.EventList{}
		opt := &client.ListOptions{}
		if err := cluster.Cache.List(ctx, opt.InNamespace(req.Namespace), events); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("get cluster[%s] events list err: %+v", cluster.GetName(), err)
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

	return as, nil
}

func applyStatus(ctx context.Context, client client.Client, req customctrl.CustomRequest, as *workloadv1beta1.AggrAppSetStatus) error {
	app := &workloadv1beta1.AppSet{}
	if err := client.Get(ctx, req.NamespacedName, app); err != nil {
		klog.V(4).Infof("%s: applyStatus get AppSet info fail: %+v", req.NamespacedName, err)
		return err
	}

	if equality.Semantic.DeepEqual(&app.Status.AggrStatus, as) {
		klog.V(4).Infof("%s: applyStatus AppSet.Status not need change", req.NamespacedName)
		return nil
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		as.DeepCopyInto(&app.Status.AggrStatus)
		t := metav1.Now()
		app.Status.LastUpdateTime = &t
		updateErr := client.Status().Update(ctx, app)
		if updateErr == nil {
			klog.V(4).Infof("%s: applyStatus update AppSet.Status success: %s", req.NamespacedName, app.Status.AggrStatus.Status)
			return nil
		}

		getErr := client.Get(ctx, req.NamespacedName, app)
		if getErr != nil {
			klog.V(4).Infof("%s: applyStatus reget AppSet info fail: %+v", req.NamespacedName, getErr)
			return getErr
		}

		klog.V(4).Infof("%s: applyStatus update AppSet.Status faile: %+v", req.NamespacedName, updateErr)
		return updateErr
	})
}
