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
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "advDeployment-controller"
)

// AdvDeploymentReconciler reconciles a AdvDeployment object
type AdvDeploymentReconciler struct {
	Name string
	client.Client
	Log     logr.Logger
	Mgr     manager.Manager
	KubeCli kubernetes.Interface
	Cfg     *rest.Config
	HelmEnv *HelmIndexSyncer
}

// func (r *AdvDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
// 	// return ctrl.NewControllerManagedBy(mgr).
// 	// 	For(&workloadv1beta1.AdvDeployment{}).
// 	// 	Owns(&appsv1.Deployment{}).
// 	// 	Owns(&appsv1.StatefulSet{}).
// 	// 	// Owns(&kruisev1alpha1.StatefulSet{}).
// 	// 	Owns(&corev1.Service{}).
// 	// 	WithOptions(controller.Options{MaxConcurrentReconciles: 3}).
// 	// 	WithEventFilter(utils.GetWatchPredicateForNs()).
// 	// 	WithEventFilter(utils.GetWatchPredicateForApp()).
// 	// 	// Watches(&source.Kind{Type: &corev1.Pod{}}, &handler.Funcs{}).
// 	// 	Watches(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: utils.GetEnqueueRequestsMapper()}).
// 	// 	Complete(r)
//
// 	return nil
// }

func Add(mgr manager.Manager, cMgr *pkgmanager.DksManager) error {
	r := &AdvDeploymentReconciler{
		Name:   "AdvDeployment-controllers",
		Client: mgr.GetClient(),
		Mgr:    mgr,
		Log:    ctrl.Log.WithName("controllers").WithName("AdvDeployment"),
	}

	r.Cfg = mgr.GetConfig()
	client, err := k8sclient.NewClientFromConfig(mgr.GetConfig())
	if err != nil {
		r.Log.Error(err, "Watch AdvDeployment err")
		return err
	}
	r.KubeCli = client

	// Create a new runtime controller
	ctl, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		r.Log.Error(err, "controller new err")
		return err
	}

	// Watch for changes to Deployment for runtime controller
	err = ctl.Watch(&source.Kind{Type: &workloadv1beta1.AdvDeployment{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		r.Log.Error(err, "Watch AdvDeployment err")
		return err
	}

	// Watch for changes to Deployment for runtime controller
	err = ctl.Watch(&source.Kind{Type: &appsv1.Deployment{}}, utils.GetEnqueueRequestsFucs(), utils.GetWatchPredicateForNs(), utils.GetWatchPredicateForApp())
	if err != nil {
		r.Log.Error(err, "Watch Deployment err")
		return err
	}

	// Watch for changes to StatefulSet for runtime controller
	err = ctl.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, utils.GetEnqueueRequestsFucs(), utils.GetWatchPredicateForNs(), utils.GetWatchPredicateForApp())
	if err != nil {
		r.Log.Error(err, "Watch StatefulSet err")
		return err
	}

	// Watch for changes to Pod for runtime controller
	err = ctl.Watch(&source.Kind{Type: &corev1.Pod{}}, utils.GetEnqueueRequestsFucs(), utils.GetWatchPredicateForNs(), utils.GetWatchPredicateForApp())
	if err != nil {
		r.Log.Error(err, "Watch Pod err")
		return err
	}

	helmv2env, err := helmv2.InitHelmRepoEnv("dmall", cMgr.Opt.Repos)
	if err != nil {
		klog.Errorf("InitHelmRepoEnv err:%v", err)
	}
	r.HelmEnv = NewDefaultHelmIndexSyncer(helmv2env)

	klog.Infof("add helm index syncer")
	mgr.Add(r.HelmEnv)
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
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		logger.Error(err, "failed to get AdvDeployment")
		return reconcile.Result{}, err
	}

	if advDeploy.ObjectMeta.DeletionTimestamp != nil {
		logger.Info("delete event", "advDeploy", advDeploy.Name)
		err := r.CleanReleasesByName(advDeploy)
		if err == nil {
			advDeploy.ObjectMeta.Finalizers = nil
			err = r.Client.Update(ctx, advDeploy)
			if err == nil {
				return reconcile.Result{}, nil
			}
		}
		return reconcile.Result{}, err
	}

	// if finalizers empty, full "sym-admin-finalizers" string
	if advDeploy.ObjectMeta.Finalizers == nil {
		advDeploy.ObjectMeta.Finalizers = []string{labels.ControllerFinalizersName}
		return reconcile.Result{}, r.Client.Update(ctx, advDeploy)
	}

	if advDeploy.Spec.PodSpec.DeployType == "helm" {
		if advDeploy.Spec.PodSpec.Chart == nil || (advDeploy.Spec.PodSpec.Chart.ChartUrl == nil && advDeploy.Spec.PodSpec.Chart.RawChart == nil) {
			klog.Errorf("name: %s DeployType is helm, but no Chart spec", advDeploy.Name)
			return reconcile.Result{}, nil
		}
	}
	// _, _ = r.reconcile(logger, advDeploy)
	// logger.Info("Reconciling AdvDeployment")
	// return ctrl.Result{
	// 	Requeue:      true,
	// 	RequeueAfter: 20 * time.Second,
	// }, nil

	return reconcile.Result{}, nil
}
