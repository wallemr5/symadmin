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
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/gofrs/uuid"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sync"
	"time"
)

const (
	controllerName = "appset-controller"
)

// Reconciler implements controller.Reconciler
type AppSetReconciler struct {
	manager.Manager
	controller.Controller
	client.Client

	DksMgr            *pkgmanager.DksManager
	SymServerRlsPath  string
	SymServerCfgPath  string
	LastReconcileTime time.Time
	MigratePeriod     time.Duration
	MigrateParallel   int
	recorder          record.EventRecorder
	Mx                sync.RWMutex
	Log               logr.Logger
	CustomImpl        *customctrl.Impl
	Namespace         string
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
		Period:           30 * time.Second,
	}
}

func (r *AppSetReconciler) PolicyEnqueueKey() {
	klog.V(4).Infof("new time: %v", time.Now())
}

// Start policy trigger loop
func (p *PolicyTrigger) Start(stop <-chan struct{}) error {
	klog.Info("start policy trigger loop ... ")
	wait.Until(p.AppSetReconciler.PolicyEnqueueKey, p.Period, stop)
	return nil
}

func NewAppSetController(mgr manager.Manager, cMgr *pkgmanager.DksManager) (*AppSetReconciler, *customctrl.Impl) {
	c := &AppSetReconciler{
		DksMgr:    cMgr,
		Manager:   mgr,
		Log:       logf.KBLog.WithName("appset-controller"),
		Namespace: "default",
	}

	// Create a new controller
	ctl, err := controller.New(controllerName, mgr, controller.Options{Reconciler: c})
	if err != nil {
		return nil, nil
	}

	// Watch for changes to AppSet
	err = ctl.Watch(&source.Kind{Type: &workloadv1beta1.AppSet{}}, &handler.EnqueueRequestForObject{})
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
	c.Client = mgr.GetClient()
	return c, customImpl
}

func (r *AppSetReconciler) CustomReconcile(ctx context.Context, req customctrl.CustomRequest) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func (r *AppSetReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("key", req.NamespacedName, "id", uuid.Must(uuid.NewV4()).String())

	as := &workloadv1beta1.AppSet{}
	err := r.Client.Get(ctx, req.NamespacedName, as)
	if err != nil {
		logger.Error(err, "failed to get AppSet", "req", req)
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	for _, v := range as.Spec.ClusterTopology.Clusters {
		if err := r.createAdvInfo(as.Name, &v); err != nil {
			// return reconcile.Result{}, err
			klog.V(3).Infof("error:%+v", err)
			return reconcile.Result{}, nil
		}
		klog.V(3).Infof("pod info:+%v", v)
	}

	klog.V(3).Infof("get appset, name: %s", as.Name)
	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: 30 * time.Second,
	}, nil
}

func (r *AppSetReconciler) createAdvInfo(info *workloadv1beta1.AppSet, clusterTopology *workloadv1beta1.TargetCluster) error {
	client, err := r.DksMgr.K8sMgr.Get(clusterTopology.Name)
	if err != nil {
		return err
	}

	advDeployment := &workloadv1beta1.AdvDeployment{}
	e := client.KubeCli.Discovery().RESTClient().
		Get().
		Namespace(r.Namespace).
		Resource(workloadv1beta1.AdvDeploymentNameStr).
		Name(info.spec.Name).
		Do().
		Into(advDeployment)

	if e != nil {
		if !apierrors.IsNotFound(e) {
			return e
		}
	}
	klog.V(3).Infof("advDeloyment info:%+v", advDeployment)

	// replicas := 0
	// advDeployment.Spec.Replicas =

	return nil

	// build AdvDeployment with AppSet cluster topology info
	// advDeployment.Spec.Replicas = clusterTopology

	// return client.KubeCli.Discovery().RESTClient().Post().
	// 	Namespace(r.Namespace).
	// 	Resource(workloadv1beta1.AdvDeploymentNameStr).
	// 	Body(advDeployment).
	// 	Do().
	// 	Error()
}
