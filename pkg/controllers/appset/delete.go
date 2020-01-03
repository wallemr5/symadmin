package appset

import (
	"context"
	"fmt"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteAll delete crd handler
func (r *AppSetReconciler) DeleteAll(ctx context.Context, req customctrl.CustomRequest, as *workloadv1beta1.AppSet) error {

	// loop cluster delete advdeployment
	for _, cluster := range r.DksMgr.K8sMgr.GetAll() {
		isChanged, err := r.DeleteByCluster(ctx, req, cluster.GetName())
		if err != nil || isChanged {
			return err
		}
	}

	// delete all cluter advdeployment info
	as.ObjectMeta.Finalizers = nil
	return r.Client.Update(ctx, as)
}

func (r *AppSetReconciler) DeleteByCluster(ctx context.Context, req customctrl.CustomRequest, clusterName string) (bool, error) {
	cluster, err := r.DksMgr.K8sMgr.Get(clusterName)
	if err != nil {
		return false, err
	}
	err = cluster.Client.Delete(ctx, &workloadv1beta1.AdvDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
	})
	if err == nil {
		return true, nil
	}
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	return false, fmt.Errorf("delete cluster:%s AdvDeployment(%s) fail:%+v", clusterName, req.NamespacedName.String(), err)
}
