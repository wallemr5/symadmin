package appset

import (
	"context"
	"fmt"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteAll delete crd handler
func (r *AppSetReconciler) DeleteAll(ctx context.Context, req customctrl.CustomRequest, as *workloadv1beta1.AppSet) error {

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

	// delete all cluter advdeployment info
	as.ObjectMeta.Finalizers = nil
	return r.Client.Update(ctx, as)
}

func deleteByCluster(ctx context.Context, cluster *k8smanager.Cluster, req customctrl.CustomRequest) (bool, error) {
	err := cluster.Client.Delete(ctx, &workloadv1beta1.AdvDeployment{
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
	return false, fmt.Errorf("delete cluster:%s AdvDeployment(%s) fail:%+v", cluster.GetName(), req.NamespacedName.String(), err)
}
