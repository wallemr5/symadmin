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

	"fmt"

	"github.com/go-logr/logr"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"

	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"
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
	Name     string
	Log      logr.Logger
	Mgr      manager.Manager
	KubeCli  kubernetes.Interface
	Cfg      *rest.Config
	Opt      *pkgmanager.ManagerOption
	recorder record.EventRecorder
}

// Add add controller to runtime manager
func Add(mgr manager.Manager, cMgr *pkgmanager.DksManager) error {
	r := &AdvDeploymentReconciler{
		Name:     controllerName,
		Client:   mgr.GetClient(),
		Mgr:      mgr,
		Log:      ctrl.Log.WithName("controllers").WithName("AdvDeployment"),
		Opt:      cMgr.Opt,
		recorder: mgr.GetEventRecorderFor(controllerName),
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

	// err = ctl.Watch(&source.Kind{Type: &v2beta2.HorizontalPodAutoscaler{}}, &handler.EnqueueRequestForOwner{OwnerType: &workloadv1beta1.AdvDeployment{}, IsController: true})
	// if err != nil {
	// 	r.Log.Error(err, "Watching HPA has an error")
	// 	return err
	// }

	// only trigger Service sync
	_, _ = mgr.GetCache().GetInformer(context.TODO(), &corev1.Service{})
	return nil
}

func (r *AdvDeploymentReconciler) DeployTypeCheck(advDeploy *workloadv1beta1.AdvDeployment) error {
	if advDeploy.Spec.PodSpec.DeployType != "helm" {
		return fmt.Errorf("advDeploy: %s not supported deploy type: %s", advDeploy.Name, advDeploy.Spec.PodSpec.DeployType)
	}

	if advDeploy.Spec.PodSpec.Chart == nil {
		return fmt.Errorf("advDeploy: %s Chart is nil", advDeploy.Name)
	}
	if advDeploy.Spec.PodSpec.Chart.ChartUrl == nil && advDeploy.Spec.PodSpec.Chart.RawChart == nil {
		return fmt.Errorf("advDeploy: %s Chart url or RawChart is nil", advDeploy.Name)
	}

	return nil
}

// +kubebuilder:rbac:groups=workload.dmall.com,resources=advdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=workload.dmall.com,resources=advdeployments/status,verbs=get;update;patch

func (r *AdvDeploymentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("key", req.NamespacedName.String())

	// Calculating how long did the reconciling process take
	startTime := time.Now()
	defer func() {
		diffTime := time.Since(startTime)
		var logLevel klog.Level
		if diffTime > 1*time.Second {
			logLevel = 2
		} else if diffTime > 100*time.Millisecond {
			logLevel = 4
		} else {
			logLevel = 5
		}
		klog.V(logLevel).Infof("##### [%s] reconciling is finished. time taken: %v. ", req.NamespacedName.String(), diffTime)
	}()

	// At first, find the advDeployment with its namespaced name.
	advDeploy := &workloadv1beta1.AdvDeployment{}
	err := r.Client.Get(ctx, req.NamespacedName, advDeploy)
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.V(3).Infof("not find any advDeploy with name: %s, skip", req.NamespacedName.String())
			return reconcile.Result{}, nil
		}

		logger.Error(err, "failed to get AdvDeployment")
		return reconcile.Result{}, err
	}

	if !advDeploy.ObjectMeta.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}

	if err := r.DeployTypeCheck(advDeploy); err != nil {
		klog.Errorf("advDeploy[%s] deploy type check failed, err: %+v", advDeploy.Name, err)
		r.recorder.Event(advDeploy, corev1.EventTypeWarning, "deploy type check failed", err.Error())
		return reconcile.Result{}, err
	}

	ownerRes, isChanged, err := r.ApplyResources(ctx, advDeploy)
	if err != nil {
		r.recorder.Event(advDeploy, corev1.EventTypeWarning, "apply resources failed", err.Error())
		logger.Error(err, "failed to apply resources")
		return reconcile.Result{}, err
	}

	if isChanged > 0 {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 5 * time.Second,
		}, nil
	}

	aggregatedStatus, isGenerationEqual, err := r.RecalculateStatus(ctx, advDeploy, ownerRes)
	if err != nil {
		klog.Errorf("advDeploy[%s] status aggregate failed, err: %+v", advDeploy.Name, err)
		r.recorder.Event(advDeploy, corev1.EventTypeWarning, "status aggregate failed", err.Error())
		return reconcile.Result{}, err
	}

	if err = r.updateStatus(ctx, advDeploy, aggregatedStatus, isGenerationEqual); err != nil {
		klog.Errorf("advDeploy[%s] status update failed, err: %+v", advDeploy.Name, err)
		r.recorder.Event(advDeploy, corev1.EventTypeWarning, "status update failed", err.Error())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
