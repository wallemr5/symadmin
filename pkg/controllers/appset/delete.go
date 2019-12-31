package appset

import (
	"context"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
)

// Delete delete crd handler
func (r *AppSetReconciler) Delete(as *workloadv1beta1.AppSet) error {
	as.ObjectMeta.Finalizers = nil
	return r.Client.Update(context.TODO(), as)
}
