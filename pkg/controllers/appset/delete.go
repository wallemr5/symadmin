package appset

import (
	"context"
	"fmt"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteAll delete crd handler
func (r *AppSetReconciler) DeleteAll(ctx context.Context, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) error {
	logger := utils.GetCtxLogger(ctx)

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

	logger.Info("delete all AdvDeployment success,delete AppSet now", "app", app.Name)
	app.ObjectMeta.Finalizers = nil
	return r.Client.Update(ctx, app)
}

func deleteByCluster(ctx context.Context, cluster *k8smanager.Cluster, req customctrl.CustomRequest) (bool, error) {
	logger := utils.GetCtxLogger(ctx)

	err := cluster.Client.Delete(ctx, &workloadv1beta1.AdvDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
	})
	if err == nil {
		logger.Info("delete Advdeployment event", "cluster", cluster.GetName())
		return true, nil
	}
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	return false, fmt.Errorf("delete cluster:%s AdvDeployment(%s) fail:%+v", cluster.GetName(), req.NamespacedName.String(), err)
}
