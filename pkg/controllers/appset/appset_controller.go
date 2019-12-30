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
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		DksMgr:    cMgr,
		Manager:   mgr,
		Log:       logf.KBLog.WithName("appset-controller"),
		Namespace: "default",
		recorder:  mgr.GetRecorder(controllerName),
	}

	// Create a new custom controller
	threadiness := 2
	customImpl := customctrl.NewImpl(c, controllerName, nil, &threadiness, labels.ObservedNamespace...)

	for _, cluster := range cMgr.K8sMgr.GetAll() {
		advDeploymentInformer, err := cluster.Cache.GetInformer(&workloadv1beta1.AdvDeployment{})
		if err != nil {
			klog.Errorf("cluster name:%s can't add AdvDeployment InformerEntry, err: %+v", cluster.Name, err)
			return nil, nil
		}
		advDeploymentInformer.AddEventHandler(customctrl.HandlerWraps(customImpl.EnqueueMulti))
		klog.Infof("cluster name:%s AddEventHandler AdvDeployment key to queue", cluster.Name)

		deploymentInformer, err := cluster.Cache.GetInformer(&appsv1.Deployment{})
		if err != nil {
			klog.Errorf("cluster name:%s can't add Deployment InformerEntry, err: %+v", cluster.Name, err)
			return nil, nil
		}
		deploymentInformer.AddEventHandler(customctrl.HandlerWraps(customImpl.EnqueueMultiLabelOfCluster))
		klog.Infof("cluster name:%s AddEventHandler Deployment key to queue", cluster.Name)

		statefulSetInformer, err := cluster.Cache.GetInformer(&appsv1.StatefulSet{})
		if err != nil {
			klog.Errorf("cluster name:%s can't add StatefulSet InformerEntry, err: %+v", cluster.Name, err)
			return nil, nil
		}
		statefulSetInformer.AddEventHandler(customctrl.HandlerWraps(customImpl.EnqueueMultiLabelOfCluster))
		klog.Infof("cluster name:%s AddEventHandler StatefulSet key to queue", cluster.Name)

		cluster.Cache.GetInformer(&corev1.Pod{})
		cluster.Cache.GetInformer(&corev1.Service{})
	}

	// Create a new runtime controller
	ctl, err := controller.New(controllerName, mgr, controller.Options{Reconciler: c})
	if err != nil {
		return nil, nil
	}

	// Watch for changes to AppSet for runtime controller
	err = ctl.Watch(&source.Kind{Type: &workloadv1beta1.AppSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return nil, nil
	}

	// Add policy trigger for same custom Enqueue
	err = mgr.Add(NewPolicyTrigger(c))
	if err != nil {
		klog.Fatal("Can't add runnable for PolicyTrigger")
	}

	c.Controller = ctl
	c.CustomImpl = customImpl
	c.Client = mgr.GetClient()
	return c, customImpl
}

// CustomReconcile for multi cluster reconcile
func (r *AppSetReconciler) CustomReconcile(ctx context.Context, req customctrl.CustomRequest) (reconcile.Result, error) {
	logger := r.Log.WithValues("cluster", req.ClusterName, "key", req.NamespacedName, "id", uuid.Must(uuid.NewV4()).String())

	as := &workloadv1beta1.AppSet{}
	err := r.Client.Get(ctx, req.NamespacedName, as)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		logger.Error(err, "failed to get AppSet")
		return reconcile.Result{}, err
	}

	for _, v := range as.Spec.ClusterTopology.Clusters {
		if err := r.createAdvInfo(as, &v, req); err != nil {
			klog.V(3).Infof("error:%+v", err)
			return reconcile.Result{}, nil
		}
		klog.V(3).Infof("pod info:%+v", v)
	}

	logger.Info("AppSet", "ResourceVersion", as.GetResourceVersion())
	return reconcile.Result{}, nil
}

// Reconcile xx
func (r *AppSetReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("key", req.NamespacedName, "id", uuid.Must(uuid.NewV4()).String())

	as := &workloadv1beta1.AppSet{}
	err := r.Client.Get(ctx, req.NamespacedName, as)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		logger.Error(err, "failed to get AppSet")
		return reconcile.Result{}, err
	}

	klog.V(3).Infof("get appset, name: %s", as.Name)
	r.CustomImpl.EnqueueKeyRateLimited(as.Namespace, as.Name, "master")

	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: 30 * time.Second,
	}, nil
}

func (r *AppSetReconciler) createAdvInfo(info *workloadv1beta1.AppSet, clusterTopology *workloadv1beta1.TargetCluster, req customctrl.CustomRequest) error {
	cluster, err := r.DksMgr.K8sMgr.Get(clusterTopology.Name)
	if err != nil {
		return err
	}

	advDeployment := &workloadv1beta1.AdvDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: info.Name,
		},
	}
	isExist := true

	err = cluster.Client.Get(context.TODO(), req.NamespacedName, advDeployment)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			klog.V(4).Infof("found err:%+v", err)
			return err
		}
		isExist = false
	}

	// build AdvDeployment info with AppSet TargetCluster
	replica := 0
	for _, v := range clusterTopology.PodSets {
		replica += v.Replicas.IntValue()
	}
	advDeployment.Spec.ServiceName = info.Spec.ServiceName
	advDeployment.Spec.PodSpec = info.Spec.PodSpec
	advDeployment.Spec.Replicas = utils.IntPointer(int32(replica))
	advDeployment.Spec.Topology = workloadv1beta1.Topology{
		PodSets: clusterTopology.PodSets,
	}
	advDeployment.Namespace = info.Namespace
	advDeployment.Name = info.Name

	if isExist {
		return cluster.Client.Update(context.TODO(), advDeployment)
	}

	return cluster.Client.Create(context.TODO(), advDeployment)
}
