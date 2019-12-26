/*
Copyright 2019 The dks authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package appset

import (
	"fmt"
	"sync"
	"time"

	"context"

	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	"k8s.io/apimachinery/pkg/util/wait"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "appset-controller"
)

// Reconciler implements controller.Reconciler
type AppSetReconciler struct {
	manager.Manager
	controller.Controller
	DksMgr            *pkgmanager.DksManager
	SymServerRlsPath  string
	SymServerCfgPath  string
	LastReconcileTime time.Time
	MigratePeriod     time.Duration
	MigrateParallel   int
	recorder          record.EventRecorder
	Mx                sync.RWMutex
	CustomImpl        *customctrl.Impl
}

func Add(mgr manager.Manager, cMgr *pkgmanager.DksManager) error {
	r, impl := NewAppSetController(mgr, cMgr)
	if r == nil {
		return fmt.Errorf("NewAppSetController err")
	}

	err := mgr.Add(impl)
	if err != nil {
		klog.Fatal("Can't add runnable for appset controller")
		return err
	}

	return nil
}

type PolicyTrigger struct {
	*AppSetReconciler
	Period time.Duration
}

func NewPolicyTrigger(r *AppSetReconciler) *PolicyTrigger {
	return &PolicyTrigger{
		AppSetReconciler: r,
		Period:           5 * time.Second,
	}
}

func (p *PolicyTrigger) Start(stop <-chan struct{}) error {
	wait.Until(p.AppSetReconciler.PolicyEnqueueKey, p.Period, stop)
	return nil
}

func NewAppSetController(mgr manager.Manager, cMgr *pkgmanager.DksManager) (*AppSetReconciler, *customctrl.Impl) {
	c := &AppSetReconciler{
		DksMgr:   cMgr,
		Manager:  mgr,
		recorder: mgr.GetEventRecorderFor(controllerName),
	}

	// cacher := mgr.GetCache()
	// appSetInformer, err := cacher.GetInformer(&workloadv1beta1.AppSet{})
	// if err != nil {
	// 	klog.Errorf("cacher get informer err:%+v", err)
	// 	return nil, nil
	// }

	// Create a new controller
	ctl, err := controller.New(controllerName, mgr, controller.Options{Reconciler: c})
	if err != nil {
		return nil, nil
	}

	// Watch for changes to UnitedDeployment
	err = c.Watch(&source.Kind{Type: &workloadv1beta1.AppSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return nil, nil
	}

	threadiness := 2
	customImpl := customctrl.NewImpl(c, controllerName, nil, &threadiness, labels.ObservedNamespace...)

	err = mgr.Add(NewPolicyTrigger(c))
	if err != nil {
		klog.Fatal("Can't add runnable for PolicyTrigger")
	}

	// cMgr.Router.AddRoutes(controllerName, c.Routes())
	c.Controller = ctl
	c.CustomImpl = customImpl
	return c, customImpl
}

func (r *AppSetReconciler) CustomReconcile(ctx context.Context, key string) error {
	return nil
}

func (r *AppSetReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func (r *AppSetReconciler) PolicyEnqueueKey() {

}
