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
	"sync"
	"time"

	"github.com/go-logr/logr"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	"gitlab.dmall.com/arch/sym-admin/pkg/healthcheck"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	controllerName = "appset-controller"
)

// Reconciler implements controller.Reconciler
type AppSetReconciler struct {
	manager.Manager
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
	klog.V(5).Infof("new time: %v", time.Now())
}

// Start policy trigger loop
func (p *PolicyTrigger) Start(stop <-chan struct{}) error {
	klog.Info("start policy trigger loop ... ")
	wait.Until(p.AppSetReconciler.PolicyEnqueueKey, p.Period, stop)
	return nil
}

func NewAppSetController(mgr manager.Manager, cMgr *pkgmanager.DksManager) (*AppSetReconciler, *customctrl.Impl) {
	c := &AppSetReconciler{
		DksMgr:   cMgr,
		Manager:  mgr,
		Log:      logf.Log.WithName(controllerName),
		recorder: mgr.GetEventRecorderFor(controllerName),
	}

	// Create a new custom controller
	customImpl := customctrl.NewImpl(c, controllerName, nil, &cMgr.Opt.Threadiness, labels.ObservedNamespace...)

	c.CustomImpl = customImpl
	c.Client = mgr.GetClient()

	for _, cluster := range cMgr.K8sMgr.GetAll() {
		err := c.registryResource(cluster)
		if err != nil {
			return nil, nil
		}
	}

	appSetInformer, err := mgr.GetCache().GetInformer(context.TODO(), &workloadv1beta1.AppSet{})
	if err != nil {
		klog.Fatalf("master appset crd informer watch err:%+v", err)
	}

	// filter owner update status
	appSetInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: customImpl.Enqueue,
		UpdateFunc: func(old, cur interface{}) {
			newObj := cur.(*workloadv1beta1.AppSet)
			oldObj := old.(*workloadv1beta1.AppSet)
			if equality.Semantic.DeepEqual(oldObj.Spec, newObj.Spec) &&
				!utils.IsObjectMetaChange(cur, old) {
				return
			}

			customImpl.Enqueue(cur)
		},

		DeleteFunc: customImpl.Enqueue,
	})

	// Add policy trigger for same custom Enqueue
	err = mgr.Add(NewPolicyTrigger(c))
	if err != nil {
		klog.Fatal("Can't add runnable for PolicyTrigger")
	}

	go c.ClusterChange()

	return c, customImpl
}

func (r *AppSetReconciler) registryResource(cluster *k8smanager.Cluster) error {
	advDeploymentInformer, err := cluster.Cache.GetInformer(context.TODO(), &workloadv1beta1.AdvDeployment{})
	if err != nil {
		klog.Errorf("cluster name:%s can't add AdvDeployment InformerEntry, err: %+v", cluster.Name, err)
		return err
	}
	advDeploymentInformer.AddEventHandler(customctrl.HandlerWraps(r.CustomImpl.EnqueueMulti))
	klog.Infof("cluster name:%s AddEventHandler AdvDeployment key to queue", cluster.Name)

	healthHandler := healthcheck.GetHealthHandler()
	healthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", cluster.Name, "advDeploy_cache_sync"), func() error {
		if advDeploymentInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s AdvDeployment cache not sync", cluster.Name)
	})

	return nil
}

func (r *AppSetReconciler) ClusterChange() {
	for {
		select {
		case list, ok := <-r.DksMgr.K8sMgr.ClusterAddInfo:
			if !ok {
				return
			}
			for name := range list {
				cluster, err := r.DksMgr.K8sMgr.Get(name)
				if err != nil {
					klog.Errorf("get cluster[%s] faile: %+v", cluster.Name, err)
					break
				}
				r.registryResource(cluster)
			}
		}
	}
}

// CustomReconcile for multi cluster reconcile
func (r *AppSetReconciler) CustomReconcile(ctx context.Context, req customctrl.CustomRequest) (reconcile.Result, error) {
	logger := r.Log.WithValues("key", req.NamespacedName)

	app := &workloadv1beta1.AppSet{}
	err := r.Client.Get(ctx, req.NamespacedName, app)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		logger.Error(err, "failed to get appSet")
		return reconcile.Result{}, err
	}

	// delete all sub Resources
	if !app.ObjectMeta.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, r.DeleteAll(ctx, req, app)
	}

	// add finalizers
	if !utils.ContainsString(app.ObjectMeta.Finalizers, labels.ControllerFinalizersName) {
		klog.V(2).Infof("%s set finalizers: %s", req.NamespacedName.String(), labels.ControllerFinalizersName)

		if app.ObjectMeta.Finalizers == nil {
			app.ObjectMeta.Finalizers = []string{}
		}
		app.ObjectMeta.Finalizers = append(app.ObjectMeta.Finalizers, labels.ControllerFinalizersName)
		err := r.Client.Update(ctx, app)
		if err != nil {
			klog.Errorf("name: %s set finalizers err: %+v", req.NamespacedName.String(), err)
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	isChanged, err := r.ApplySpec(ctx, req, app)
	if err != nil {
		logger.Error(err, "apply advdeployment info with spec")
		r.recorder.Event(app, corev1.EventTypeWarning, "apply advDeployment failed", err.Error())
		return reconcile.Result{}, err
	}

	if isChanged > 0 {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	klog.V(5).Infof("%s start aggregate status ... ", req.NamespacedName.String())
	status, _, err := r.ApplyStatus(ctx, req, app)
	if err != nil {
		logger.Error(err, "update AppSet Status fail")
		return reconcile.Result{}, err
	}

	_, err = r.DeleteUnuseAdvDeployment(ctx, req, status)
	if err != nil {
		logger.Error(err, "delete unexpect info")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
