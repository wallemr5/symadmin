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
	"time"

	"github.com/go-logr/logr"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	"gitlab.dmall.com/arch/sym-admin/pkg/helm/v2repo"
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "advDeployment-controller"
)

// AdvDeploymentReconciler reconciles a AdvDeployment object
type AdvDeploymentReconciler struct {
	client.Client

	Name      string
	Log       logr.Logger
	Mgr       manager.Manager
	KubeCli   kubernetes.Interface
	Cfg       *rest.Config
	HelmEnv   *v2repo.HelmIndexSyncer
	IsRecover bool
	recorder  record.EventRecorder
}

// Add add controller to runtime manager
func Add(mgr manager.Manager, cMgr *pkgmanager.DksManager) error {
	r := &AdvDeploymentReconciler{
		Name:      controllerName,
		Client:    mgr.GetClient(),
		Mgr:       mgr,
		Log:       ctrl.Log.WithName("controllers").WithName("AdvDeployment"),
		IsRecover: cMgr.Opt.Recover,
		recorder:  mgr.GetRecorder(controllerName),
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
	r.HelmEnv = v2repo.NewDefaultHelmIndexSyncer(helmv2env)

	klog.Infof("add helm repo index syncer Runnable")
	mgr.Add(r.HelmEnv)
	return nil
}

// +kubebuilder:rbac:groups=workload.dmall.com,resources=advdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workload.dmall.com,resources=advdeployments/status,verbs=get;update;patch

func (r *AdvDeploymentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	klog.V(3).Infof("##### [%s] start to reconcile.", req.NamespacedName)

	ctx := context.Background()
	logger := r.Log.WithValues("key", req.NamespacedName, "id", uuid.Must(uuid.NewV4()).String())

	// Calculating how long did the reconciling process take
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
		klog.V(logLevel).Infof("##### [%s] reconciling is finished. time taken: %v. ", req.NamespacedName, diffTime)
	}()

	// exec recover logic
	if r.IsRecover {
		return r.Recover(req)
	}

	// At first, find the advDeployment with its namespaced name.
	advDeploy := &workloadv1beta1.AdvDeployment{}
	err := r.Client.Get(ctx, req.NamespacedName, advDeploy)
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.V(3).Infof("Can not find any advDeploy with name [%s], don't care about it.", req.NamespacedName)
			return reconcile.Result{}, nil
		}

		logger.Error(err, "failed to get AdvDeployment")
		return reconcile.Result{}, err
	}

	/*
		Before delete AdvDeployment, we will clean all installed helm releases,
		if the deletionTimestamp is nil, it means this advDeploy has been deleted by someone.
		it also mean that we have to delete all releases of this advDeploy immediately.
	*/
	if !advDeploy.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.Info("Delete all releases of an advDeploy", "advDeploy", advDeploy.Name)
		if err := r.CleanAllReleases(advDeploy); err != nil {
			logger.Error(err, "Can not remove the helm releases which are related with this AdvDeployment")
			r.recorder.Event(advDeploy, corev1.EventTypeWarning, "Remove helm release failed", err.Error())
			return reconcile.Result{}, errors.Wrap(err, "Can not remove the helm releases which are related with this AdvDeployment")
		}

		return reconcile.Result{}, r.RemoveFinalizers(ctx, req)
	}

	// If you found that its finalizer list is empty or nil, we must append sym-admin's finalizer into this list if the deletionTimeStamp is nil.
	if !utils.ContainsString(advDeploy.ObjectMeta.Finalizers, labels.ControllerFinalizersName) {
		// This list may be nil if sym-admin is not its owner.
		if advDeploy.ObjectMeta.Finalizers == nil {
			advDeploy.ObjectMeta.Finalizers = []string{}
		}

		advDeploy.ObjectMeta.Finalizers = append(advDeploy.ObjectMeta.Finalizers, labels.ControllerFinalizersName)
		if err := r.Client.Update(ctx, advDeploy); err != nil {
			klog.Errorf("failed to update AdvDeployment[%s] for appending a finalizer to it: %v", advDeploy.Name, err)
			r.recorder.Event(advDeploy, corev1.EventTypeWarning, "Add finalizer failed", err.Error())
			return reconcile.Result{}, errors.Wrap(err, "Can not add sym-admin's finalizer to AdvDeployment")
		}

		klog.V(3).Infof("Adding the finalizer to advDeploy[%s] successfully", advDeploy.Name)
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: time.Second * 5,
		}, nil
	}

	// Converge the releases
	// So far the advDeployment is not deleted by others, we must keep it right.
	if advDeploy.Spec.PodSpec.DeployType == "helm" {
		if advDeploy.Spec.PodSpec.Chart == nil {
			klog.Errorf("advDeploy [%s]'s chart is empty, can not reconcile it.", advDeploy.Name)
			r.recorder.Event(advDeploy, corev1.EventTypeWarning, "Chart is empty", "chart is empty, can not reconcile it.")
			return reconcile.Result{}, nil
		}
		if advDeploy.Spec.PodSpec.Chart.ChartUrl == nil && advDeploy.Spec.PodSpec.Chart.RawChart == nil {
			klog.Errorf("advDeploy [%s]: neither chart's url nor raw exist, can not reconcile it.", advDeploy.Name)
			r.recorder.Event(advDeploy, corev1.EventTypeWarning, "Chart data not neither empty", "neither chart's url nor raw exist, can not reconcile it.")
			return reconcile.Result{}, nil
		}

		_, err := r.ApplyReleases(ctx, advDeploy)
		if err != nil {
			r.recorder.Event(advDeploy, corev1.EventTypeWarning, "Apply releases failed", err.Error())
			logger.Error(err, "failed to apply releases")
		}

		//if hasModifiedRls {
		//	return reconcile.Result{}, err
		//}
	} else {
		klog.Errorf("The deploy type %s don't be supported yet.", advDeploy.Name)
	}

	// We can update the status for the advDeployment without modification for any release.
	aggregatedStatus, err := r.RecalculateStatus(ctx, advDeploy)
	if err != nil {
		klog.Errorf("failed to recalculate the newest status of an advancement deployment [%s]: %v", advDeploy.Name, err)
		r.recorder.Event(advDeploy, corev1.EventTypeWarning, "Aggregate status failed", err.Error())
		return reconcile.Result{}, err
	}

	if err := r.updateStatus(ctx, advDeploy, aggregatedStatus); err != nil {
		klog.Errorf("failed to update the newest status of an advancement deployment [%s]: %v", advDeploy.Name, err)
		r.recorder.Event(advDeploy, corev1.EventTypeWarning, "Update newest status failed", err.Error())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
