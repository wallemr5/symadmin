package appset

import (
	"context"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Recover ...
func (r *AppSetReconciler) Recover(ctx context.Context, req customctrl.CustomRequest) (reconcile.Result, error) {

	app := &workloadv1beta1.AppSet{}
	isCreate, err := r.buildAppSet(ctx, req, app)
	if err != nil {
		return reconcile.Result{}, err
	}

	if isCreate {
		err := r.Client.Create(context.TODO(), app)
		if err != nil {
			klog.Errorf("Create AppSet failed: %s", err.Error())
		}
		return reconcile.Result{}, err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {

		updateErr := r.Client.Update(context.TODO(), app)
		if updateErr == nil {
			klog.V(4).Info("Update AppSet success")
			return nil
		}

		_, getErr := r.buildAppSet(ctx, req, app)
		if getErr != nil {
			klog.Errorf("Getting updated AppSet failed: %s", getErr.Error())
			return getErr
		}
		return updateErr
	})

	return reconcile.Result{}, err
}

func (r *AppSetReconciler) buildAppSet(ctx context.Context, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) (isCreate bool, err error) {
	app = &workloadv1beta1.AppSet{}
	err = r.GetClient().Get(ctx, req.NamespacedName, app)
	if err != nil && !apierrors.IsNotFound(err) {
		if apierrors.IsNotFound(err) {
			isCreate = true
		} else {
			klog.Errorf("Recover get Appset err:%s", err.Error())
			return false, err
		}
	}

	advList := []*workloadv1beta1.AdvDeployment{}
	for _, cluster := range r.DksMgr.K8sMgr.GetAll() {
		adv := &workloadv1beta1.AdvDeployment{}
		err = cluster.Client.Get(ctx, req.NamespacedName, adv)
		if err != nil {
			klog.Errorf("Get [%s] AdvDeployment failed: %s", cluster.Name, err.Error())
			continue
		}
		advList = append(advList, adv)
	}

	if isCreate {
		app.ObjectMeta.Name = req.Name
		app.ObjectMeta.Namespace = req.Namespace
		app.Spec.ServiceName = advList[0].Spec.ServiceName

		advList[0].Spec.PodSpec.DeepCopyInto(&app.Spec.PodSpec)
	}
	app.Spec.ClusterTopology.Clusters = []*workloadv1beta1.TargetCluster{}
	for _, adv := range advList {
		app.Spec.ClusterTopology.Clusters = append(app.Spec.ClusterTopology.Clusters, &workloadv1beta1.TargetCluster{
			Name:    req.ClusterName,
			Mata:    adv.Labels,
			PodSets: adv.Spec.Topology.PodSets,
		})
	}

	var replica int32
	for _, cluster := range app.Spec.ClusterTopology.Clusters {
		for _, ps := range cluster.PodSets {
			replica += ps.Replicas.IntVal
		}
	}
	app.Spec.Replicas = &replica

	return isCreate, nil
}
