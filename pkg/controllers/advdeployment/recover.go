package advdeployment

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *AdvDeploymentReconciler) Recover(req ctrl.Request) (ctrl.Result, error) {
	return reconcile.Result{}, nil
}
