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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"time"

	"github.com/pkg/errors"
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

// Add add controller to runtime manager
func Add(mgr manager.Manager, cMgr *pkgmanager.DksManager) error {
	r := &AdvDeploymentReconciler{
		Name:   "AdvDeployment-controllers",
		Client: mgr.GetClient(),
		Mgr:    mgr,
		Log:    ctrl.Log.WithName("controllers").WithName("AdvDeployment"),
	}

	r.Cfg = mgr.GetConfig()
	kubeCli, err := k8sclient.NewClientFromConfig(mgr.GetConfig())
	if err != nil {
		r.Log.Error(err, "Creating a kube client for the reconciler has an error")
		return err
	}
	r.KubeCli = kubeCli

	// Create a new runtime controller for advDeployment
	ctl, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r, MaxConcurrentReconciles: cMgr.Opt.Threadiness})
	if err != nil {
		r.Log.Error(err, "Creating a new AdvDeployment controller has an error")
		return err
	}

	// We set the objects which would to be watched by this controller.
	err = ctl.Watch(&source.Kind{Type: &workloadv1beta1.AdvDeployment{}}, &handler.EnqueueRequestForObject{}, utils.GetWatchPredicateForAdvDeploymentSpec())
	if err != nil {
		r.Log.Error(err, "Watching AdvDeployment has an error")
		return err
	}

	// Watch for changes to Deployment for runtime controller
	err = ctl.Watch(&source.Kind{Type: &appsv1.Deployment{}}, utils.GetEnqueueRequestsFucs(), utils.GetWatchPredicateForNs(), utils.GetWatchPredicateForApp())
	if err != nil {
		r.Log.Error(err, "Watching Deployment has an error")
		return err
	}

	// Watch for changes to StatefulSet for runtime controller
	err = ctl.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, utils.GetEnqueueRequestsFucs(), utils.GetWatchPredicateForNs(), utils.GetWatchPredicateForApp())
	if err != nil {
		r.Log.Error(err, "Watching StatefulSet has an error")
		return err
	}

	// // Watch for changes to Pod for runtime controller
	// err = ctl.Watch(&source.Kind{Type: &corev1.Pod{}}, utils.GetEnqueueRequestsFucs(), utils.GetWatchPredicateForNs(), utils.GetWatchPredicateForApp())
	// if err != nil {
	// 	r.Log.Error(err, "Watch Pod err")
	// 	return err
	// }

	helmv2env, err := helmv2.InitHelmRepoEnv("dmall", cMgr.Opt.Repos)
	if err != nil {
		klog.Errorf("Initializing a helm env has an error:%v", err)
	}
	r.HelmEnv = NewDefaultHelmIndexSyncer(helmv2env)

	klog.Infof("add helm repo index syncer Runnable")
	mgr.Add(r.HelmEnv)
	return nil
}

// +kubebuilder:rbac:groups=workload.dmall.com,resources=advdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workload.dmall.com,resources=advdeployments/status,verbs=get;update;patch

func (r *AdvDeploymentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("key", req.NamespacedName, "id", uuid.Must(uuid.NewV4()).String())
	startTime := time.Now()
	defer func() {
		diffTime := time.Since(startTime)
		var logLevel klog.Level
		if diffTime > 2*time.Second {
			logLevel = 2
		} else if diffTime > 1*time.Second {
			logLevel = 3
		} else {
			logLevel = 4
		}
		klog.V(logLevel).Infof("key: %v Reconcile end. Time taken: %v. ", req, diffTime)
	}()

	advDeploy := &workloadv1beta1.AdvDeployment{}
	err := r.Client.Get(ctx, req.NamespacedName, advDeploy)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		logger.Error(err, "failed to get AdvDeployment")
		return reconcile.Result{}, err
	}

	// before delete AdvDeployment, we will clean all installed helm releases
	if !advDeploy.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.Info("delete event", "advDeploy", advDeploy.Name)
		if err := r.CleanReleasesByName(advDeploy); err != nil {
			return reconcile.Result{}, errors.Wrap(err, "could not remove helm release to AdvDeployment")
		}
		klog.V(3).Infof("advDeploy: %s clean all helm Releases success, than update Finalizers nil", advDeploy.Name)
		advDeploy.ObjectMeta.Finalizers = []string{}
		if err := r.Client.Update(ctx, advDeploy); err != nil {
			return reconcile.Result{}, errors.Wrap(err, "could not remove finalizer to AdvDeployment")
		}
		klog.V(3).Infof("advDeploy: %s Update Finalizers nil success", advDeploy.Name)
		return reconcile.Result{}, nil
	}

	// if finalizers empty, full ControllerFinalizersName string
	if len(advDeploy.ObjectMeta.Finalizers) == 0 {
		advDeploy.ObjectMeta.Finalizers = []string{labels.ControllerFinalizersName}
		if err := r.Client.Update(ctx, advDeploy); err != nil {
			logger.Error(err, "failed to get AdvDeployment")
			return reconcile.Result{}, errors.Wrap(err, "could not add finalizer to AdvDeployment")
		}
		klog.V(3).Infof("advDeploy: %s Update add Finalizers success")
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 5,
		}, nil
	}

	// at present, wo only process deploy type is helm
	if advDeploy.Spec.PodSpec.DeployType == "helm" {
		if advDeploy.Spec.PodSpec.Chart == nil || (advDeploy.Spec.PodSpec.Chart.ChartUrl == nil && advDeploy.Spec.PodSpec.Chart.RawChart == nil) {
			klog.Errorf("name: %s DeployType is helm, but no Chart spec", advDeploy.Name)
			return reconcile.Result{}, nil
		}

		err := r.ApplyPodSetReleases(ctx, advDeploy)
		if err != nil {
			logger.Error(err, "faild ApplyPodSetReleases")
			return reconcile.Result{}, err
		}
	}

	aggrStatus, err := r.RecalculateAppSetStatus(ctx, advDeploy)
	if err != nil {
		logger.Error(err, "faild RecalculateAppSetStatus")
		return reconcile.Result{}, err
	}

	if err := r.updateAggrStatus(ctx, advDeploy, aggrStatus); err != nil {
		logger.Error(err, "faild updateAggrStatus")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
