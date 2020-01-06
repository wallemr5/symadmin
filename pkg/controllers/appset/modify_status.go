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
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ModifyStatus modify status handler
func (r *AppSetReconciler) ModifyStatus(ctx context.Context, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) error {
	as, err := buildAppSetStatus(ctx, r.DksMgr.K8sMgr, req)
	if err != nil {
		klog.V(4).Infof("%s:aggregate AppSet.Status faile:%+v", req.NamespacedName, err)
		return err
	}

	klog.V(4).Infof("%s:aggregate AppSet.Status info:%+v", req.NamespacedName, as)
	if err := applyStatus(ctx, r.Client, req, as); err != nil {
		return err
	}
	return nil
}

func buildAppSetStatus(ctx context.Context, dksManger *k8smanager.ClusterManager, req customctrl.CustomRequest) (*workloadv1beta1.AggrAppSetStatus, error) {
	as := &workloadv1beta1.AggrAppSetStatus{
		Clusters: []*workloadv1beta1.ClusterAppActual{},
	}

	for _, cluster := range dksManger.GetAllSort() {
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
		if !strings.Contains(as.Version, obj.Status.AggrStatus.Version) {
			as.Version = obj.Status.AggrStatus.Version
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
			return nil, fmt.Errorf("get cluster:%s events list err:%+v", cluster.GetName(), err)
		}

		// aggregate events
		var aevt evtlist = []*workloadv1beta1.Event{}
		for _, evt := range events.Items {
			if strings.HasPrefix(evt.InvolvedObject.Name, req.Name) && evt.Type == corev1.EventTypeWarning {
				aevt = append(aevt, &workloadv1beta1.Event{
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
		sort.Sort(aevt)

		if as.WarnEvents == nil {
			as.WarnEvents = []*workloadv1beta1.Event{}
		}
		as.WarnEvents = aevt
	}

	return as, nil
}

type evtlist []*workloadv1beta1.Event

func (e evtlist) Len() int {
	return len(e)
}

func (e evtlist) Less(i, j int) bool {
	return e[i].Name > e[j].Name && e[i].Reason > e[j].Reason
}

func (e evtlist) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func applyStatus(ctx context.Context, client client.Client, req customctrl.CustomRequest, as *workloadv1beta1.AggrAppSetStatus) error {
	app := &workloadv1beta1.AppSet{}
	if err := client.Get(ctx, req.NamespacedName, app); err != nil {
		return err
	}

	if equality.Semantic.DeepEqual(&app.Status.AggrStatus, as) {
		klog.V(4).Infof("%s:AppSet.Status not need change", req.NamespacedName)
		return nil
	}

	as.DeepCopyInto(&app.Status.AggrStatus)
	t := metav1.Now()
	app.Status.LastUpdateTime = &t
	if err := client.Status().Update(ctx, app); err != nil {
		klog.V(4).Infof("%s:update AppSet.Status faile:%+v", req.NamespacedName, err)
		return err
	}
	klog.V(4).Infof("%s:update AppSet.Status success", req.NamespacedName)
	return nil
}
