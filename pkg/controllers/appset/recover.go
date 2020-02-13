package appset

import (
	"context"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *AppSetReconciler) Recover(ctx context.Context, req customctrl.CustomRequest) (reconcile.Result, error) {

	cli, err := r.DksMgr.K8sMgr.Get(req.ClusterName)
	if err != nil {
		return reconcile.Result{}, err
	}
	adv := &workloadv1beta1.AdvDeployment{}
	if err := cli.Cache.Get(ctx, req.NamespacedName, adv); err != nil {
		klog.Errorf("Get [%s] AdvDeployment failed: %s", req.ClusterName, err.Error())
		return reconcile.Result{}, err
	}

	app := &workloadv1beta1.AppSet{}
	err = r.GetClient().Get(ctx, req.NamespacedName, app)
	if err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, err
	}

	var isCreate bool
	if err == nil {
		isCreate = true
	}

	return reconcile.Result{}, apply(r.GetClient(), req, app, adv, isCreate)
}

func apply(cli client.Client, req customctrl.CustomRequest, app *workloadv1beta1.AppSet, adv *workloadv1beta1.AdvDeployment, isCreate bool) error {

	buildAppSet(req, app, adv, isCreate)

	if isCreate {
		err := cli.Create(context.TODO(), app)
		if err != nil {
			klog.Error("Create AppSet failed:%s", err.Error())
		}
		return err
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {

		updateErr := cli.Update(context.TODO(), app)
		if updateErr == nil {
			klog.V(4).Info("Update AppSet success")
			return nil
		}

		getErr := cli.Get(context.TODO(), req.NamespacedName, app)
		if getErr != nil {
			klog.Errorf("Getting updated AppSet failed: %s", getErr.Error())
			return getErr
		}
		buildAppSet(req, app, adv, isCreate)
		return updateErr
	})
}

func buildAppSet(req customctrl.CustomRequest, app *workloadv1beta1.AppSet, adv *workloadv1beta1.AdvDeployment, isCreate bool) {

	if isCreate {
		app.ObjectMeta.Name = adv.ObjectMeta.Name
		app.ObjectMeta.Namespace = adv.ObjectMeta.Namespace

		app.Spec.ClusterTopology.Clusters = []*workloadv1beta1.TargetCluster{}
		app.Spec.ClusterTopology.Clusters = append(app.Spec.ClusterTopology.Clusters, &workloadv1beta1.TargetCluster{
			Name:    req.ClusterName,
			Mata:    adv.Labels,
			PodSets: adv.Spec.Topology.PodSets,
		})
	} else {
		for i := range app.Spec.ClusterTopology.Clusters {
			if app.Spec.ClusterTopology.Clusters[i].Name == req.ClusterName {
				app.Spec.ClusterTopology.Clusters[i] = &workloadv1beta1.TargetCluster{
					Name:    req.ClusterName,
					Mata:    adv.Labels,
					PodSets: adv.Spec.Topology.PodSets,
				}
			}
		}
	}

	app.Spec.ServiceName = adv.Spec.ServiceName
	adv.Spec.PodSpec.DeepCopyInto(&app.Spec.PodSpec)

	var replica int
	for _, cluster := range app.Spec.ClusterTopology.Clusters {
		for _, ps := range cluster.PodSets {
			replica += ps.Replicas.IntValue()
		}
	}
	app.Spec.Replicas = utils.IntPointer(int32(replica))
}
