package eventwatcher

import (
	"github.com/go-logr/logr"
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"
	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "events-controller"
)

type eventReconciler struct {
	client.Client
	Name    string
	Log     logr.Logger
	Mgr     manager.Manager
	KubeCli kubernetes.Interface
	Cfg     *rest.Config
}

func Add(mgr manager.Manager, cMgr *pkgmanager.DksManager) error {
	r := &eventReconciler{
		Name:   controllerName,
		Client: mgr.GetClient(),
		Mgr:    mgr,
		Log:    ctrl.Log.WithName("controllers").WithName("events"),
	}

	r.Cfg = mgr.GetConfig()
	kubeCli, err := k8sclient.NewClientFromConfig(mgr.GetConfig())
	if err != nil {
		r.Log.Error(err, "Creating a kube client for the reconciler has an error")
		return err
	}
	r.KubeCli = kubeCli

	// Create a new runtime controller for events
	ctl, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r, MaxConcurrentReconciles: cMgr.Opt.Threadiness})
	if err != nil {
		r.Log.Error(err, "Creating a new event controller has an error")
		return err
	}

	// Watch for changes to events for runtime controller
	err = ctl.Watch(&source.Kind{Type: &corev1.Event{}}, utils.GetEnqueueRequestsFucs())
	if err != nil {
		r.Log.Error(err, "Watching event has an error")
		return err
	}

	return nil
}
func (r *eventReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return reconcile.Result{}, nil
}
