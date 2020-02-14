package advdeployment

import (
	"context"
	"strings"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *AdvDeploymentReconciler) Recover(req ctrl.Request) (ctrl.Result, error) {
	// judge name
	appInfo, ok := labels.CheckAndGetAppInfo(req.Name)
	if !ok {
		return reconcile.Result{}, nil
	}

	adv := &workloadv1beta1.AdvDeployment{}
	isCreate, err := r.buildAdvDeployment(req, appInfo, adv)
	if err != nil {
		return reconcile.Result{}, nil
	}
	if isCreate {
		err = r.Client.Create(context.TODO(), adv)
		if err != nil {
			klog.Errorf("Create AdvDeployment failed:%s", err.Error())
		}
		return reconcile.Result{}, err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		updateErr := r.Client.Update(context.TODO(), adv)
		if updateErr == nil {
			klog.V(4).Info("Update AdvDeployment success")
			return nil
		}

		_, getErr := r.buildAdvDeployment(req, appInfo, adv)
		if getErr != nil {
			klog.Errorf("Getting updated AdvDeployment failed: %s", getErr.Error())
			return getErr
		}
		return updateErr
	})

	return reconcile.Result{}, err
}

func (r *AdvDeploymentReconciler) buildAdvDeployment(req reconcile.Request, appInfo labels.AppInfo, adv *workloadv1beta1.AdvDeployment) (isCreate bool, err error) {

	err = r.Client.Get(context.TODO(), req.NamespacedName, adv)
	if err != nil && apierrors.IsNotFound(err) {
		klog.Errorf("Recover get AdvDeployment err:%s", err.Error())
		return false, err
	}
	if err == nil {
		isCreate = true
	}

	if isCreate {
		adv.Spec.Topology.PodSets = []*workloadv1beta1.PodSet{}

		adv.ObjectMeta.Name = appInfo.Name
		adv.ObjectMeta.Namespace = req.Namespace
	}

	// get deployment info
	opts := &client.ListOptions{}
	opts.MatchingLabels(map[string]string{
		"app": req.Name,
	})
	deployLists := appsv1.DeploymentList{}
	err = r.Client.List(context.TODO(), opts, &deployLists)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return
		}

		klog.Errorf("failed to DeploymentList name:%s, err: %v", req.Name, err)
		return
	}
	for _, deploy := range deployLists.Items {
		containers := deploy.Spec.Template.Spec.Containers
		if len(containers) == 0 {
			continue
		}
		info, ok := labels.CheckAndGetAppInfo(deploy.ObjectMeta.Name)
		if !ok {
			continue
		}

		images := strings.Split(containers[0].Image, ":")
		version := ""
		if len(images) > 1 {
			version = images[len(images)-1]
		}
		adv.Spec.Topology.PodSets = append(adv.Spec.Topology.PodSets, &workloadv1beta1.PodSet{
			Name:     deploy.ObjectMeta.Name,
			Replicas: &intstr.IntOrString{IntVal: *deploy.Spec.Replicas},
			Image:    images[0],
			Version:  version,
			Mata: map[string]string{
				"sym-ldc":   info.IdcName,
				"sym-group": info.Group,
			},
		})
	}

	// get helm release
	// hClient, err := helmv2.NewClientFromConfig(r.Cfg, r.KubeCli, "")
	// if err != nil {
	// 	klog.Errorf("Initializing a new helm client has an error:%s", err.Error())
	// 	return reconcile.Result{}, err
	// }
	// defer hClient.Close()

	// response, err := helmv2.ListReleases(labels.MakeHelmReleaseFilter(appInfo.Name), hClient)
	// if err != nil {
	// 	klog.Errorf("Can not find all releases for Deployment[%s], err:%s", appInfo.Name, err.Error())
	// 	return reconcile.Result{}, err
	// }
	// for _, item := range response.Releases {
	// 	// item.GetChart().
	// 	// item.String()
	// 	item.GetChart().GetTemplates()
	// }

	var replicas int32
	for _, podSet := range adv.Spec.Topology.PodSets {
		replicas += podSet.Replicas.IntVal
	}
	adv.Spec.Replicas = &replicas

	return isCreate, nil
}
