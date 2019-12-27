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

package advdeployment

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources/deployment"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources/svc"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	helmenv "k8s.io/helm/pkg/helm/environment"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// "sigs.k8s.io/controller-runtime/pkg/controller"
	// "sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	// "sigs.k8s.io/controller-runtime/pkg/source"
)

// AdvDeploymentReconciler reconciles a AdvDeployment object
type AdvDeploymentReconciler struct {
	Name string
	client.Client
	Log       logr.Logger
	Mgr       manager.Manager
	Helmv2env *helmenv.EnvSettings
}

func (r *AdvDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// return ctrl.NewControllerManagedBy(mgr).
	// 	For(&workloadv1beta1.AdvDeployment{}).
	// 	Owns(&appsv1.Deployment{}).
	// 	Owns(&appsv1.StatefulSet{}).
	// 	// Owns(&kruisev1alpha1.StatefulSet{}).
	// 	Owns(&corev1.Service{}).
	// 	WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
	// 	WithEventFilter(utils.GetWatchPredicateForNs()).
	// 	WithEventFilter(utils.GetWatchPredicateForApp()).
	// 	// Watches(&source.Kind{Type: &corev1.Pod{}}, &handler.Funcs{}).
	// 	Watches(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: utils.GetEnqueueRequestsMapper()}).
	// 	Complete(r)

	return nil
}

func Add(mgr manager.Manager, cMgr *pkgmanager.DksManager) error {
	reconciler := &AdvDeploymentReconciler{
		Name:   "AdvDeployment-controllers",
		Client: mgr.GetClient(),
		Mgr:    mgr,
		Log:    ctrl.Log.WithName("controllers").WithName("AdvDeployment"),
	}

	err := reconciler.SetupWithManager(mgr)
	if err != nil {
		return errors.Wrapf(err, "unable to create AdvDeployment controller")
	}

	helmv2env, err := helmv2.InitHelmRepoEnv("dmall", cMgr.Opt.Repos)
	if err != nil {
		klog.Errorf("InitHelmRepoEnv err:%v", err)
	}
	reconciler.Helmv2env = helmv2env
	return nil
}

// +kubebuilder:rbac:groups=workload.dmall.com,resources=advdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workload.dmall.com,resources=advdeployments/status,verbs=get;update;patch

func (r *AdvDeploymentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("key", req.NamespacedName, "id", uuid.Must(uuid.NewV4()).String())

	advDeploy := &workloadv1beta1.AdvDeployment{}
	err := r.Client.Get(ctx, req.NamespacedName, advDeploy)
	if err != nil {
		logger.Error(err, "failed to get AdvDeployment")
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	rSet := &appsv1.ReplicaSet{}

	err = r.Client.Get(ctx, types.NamespacedName{Name: "sym-operator-5d8f9f7dcc", Namespace: "sym"}, rSet)
	if err != nil {
		logger.Error(err, "failed to get ReplicaSet")
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	podall := make([]*corev1.Pod, 0, 8)

	pods := &corev1.PodList{}
	err = r.Client.List(ctx, &client.ListOptions{Namespace: "dmall-innner"}, pods)
	if err != nil {
		logger.Error(err, "failed to get ReplicaSet")
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	for i := range pods.Items {
		podall = append(podall, &pods.Items[i])
	}

	events := &corev1.EventList{}
	// err = r.Client.List(ctx, events)
	if err != nil {
		logger.Error(err, "failed to get events")
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}
	logger.Info("Reconciling get events", "num", len(events.Items))

	// _, _ = r.reconcile(logger, advDeploy)
	// logger.Info("Reconciling AdvDeployment")
	// return ctrl.Result{
	// 	Requeue:      true,
	// 	RequeueAfter: 20 * time.Second,
	// }, nil

	return reconcile.Result{}, nil
}

func (r *AdvDeploymentReconciler) reconcile(logger logr.Logger, config *workloadv1beta1.AdvDeployment) (reconcile.Result, error) {
	reconcilers := []resources.ComponentReconciler{
		svc.New(r.Mgr, config, 8080),
		deployment.New(r.Mgr, config),
	}

	for _, rec := range reconcilers {
		err := rec.Reconcile(logger)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	logger.Info("reconcile finished")
	return reconcile.Result{}, nil
}
