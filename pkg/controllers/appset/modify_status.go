package appset

import (
	"context"
	"fmt"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// ModifyStatus modify status handler
func (r *AppSetReconciler) ModifyStatus() {

}

func (r *AppSetReconciler) buildAppSetStatus(ctx context.Context, req customctrl.CustomRequest) (*workloadv1beta1.AppSet, error) {
	// status := &workloadv1beta1.AdvDeploymentStatus{}
	// obj.Status.AppActual
	for _, cluster := range r.DksMgr.K8sMgr.GetAll() {
		advDeployment := &workloadv1beta1.AdvDeployment{}
		if err := cluster.Cache.Get(ctx, req.NamespacedName, advDeployment); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("get cluster:%s AdvDeployment info err:%+v", cluster.GetName(), err)
		}
	}
	return nil, nil
}
