package advdeployment

import (
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// GetStatefulSetByLabels Finding all statefuls with a label
func (r *AdvDeploymentReconciler) GetStatefulSetByLabels(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment) ([]*appsv1.StatefulSet, error) {
	opts := &client.ListOptions{}
	opts.MatchingLabels(map[string]string{
		"app": advDeploy.Name,
	})
	staSets := appsv1.StatefulSetList{}
	err := r.Client.List(ctx, opts, &staSets)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, err
		}

		klog.Errorf("failed to StatefulSetList name:%s, err: %v", advDeploy.Name, err)
		return nil, err
	}

	statefulSets := make([]*appsv1.StatefulSet, 0, 4)
	for i := range staSets.Items {
		statefulSets = append(statefulSets, &staSets.Items[i])
	}
	return statefulSets, nil
}

// GetDeployListByByLabels Finding all deployments with a label
func (r *AdvDeploymentReconciler) GetDeployListByByLabels(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment) ([]*appsv1.Deployment, error) {
	opts := &client.ListOptions{}
	opts.MatchingLabels(map[string]string{
		"app": advDeploy.Name,
	})

	deployLists := appsv1.DeploymentList{}
	err := r.Client.List(ctx, opts, &deployLists)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, err
		}

		klog.Errorf("failed to DeploymentList name:%s, err: %v", advDeploy.Name, err)
		return nil, err
	}

	deploys := make([]*appsv1.Deployment, 0, 4)
	for i := range deployLists.Items {
		deploys = append(deploys, &deployLists.Items[i])
	}

	return deploys, nil
}

// GetServiceByByLabels Finding all services with a label
func (r *AdvDeploymentReconciler) GetServiceByByLabels(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment) (*corev1.Service, error) {
	listOptions := &client.ListOptions{}
	listOptions.MatchingLabels(map[string]string{
		"app": advDeploy.Name + "-svc",
	})

	svcLists := corev1.ServiceList{}
	err := r.Client.List(ctx, listOptions, &svcLists)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, err
		}

		klog.Errorf("failed to get svcs advDeploy name:%s, err: %v", advDeploy.Name, err)
		return nil, err
	}

	if len(svcLists.Items) != 1 {
		return nil, fmt.Errorf("advDeploy name: %s, have more or less svc len:%d", advDeploy.Name, len(svcLists.Items))
	} else {
		return &svcLists.Items[0], nil
	}
}

func IsUnUseObject(kind string, obj Object, ownerRes []string) bool {
	name := GetFormattedName(kind, obj)
	for i := range ownerRes {
		if name == ownerRes[i] {
			return false
		}
	}

	return true
}

// RecalculateStatus According to the status of running deployments, calculate the advDeployment's status
func (r *AdvDeploymentReconciler) RecalculateStatus(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment, ownerRes []string) (*workloadv1beta1.AdvDeploymentAggrStatus, error) {
	deploys, err := r.GetDeployListByByLabels(ctx, advDeploy)
	if err != nil {
		return nil, errors.Wrapf(err, "Find all relative deployments with application name [%s] has an error", advDeploy.Name)
	}

	svc, err := r.GetServiceByByLabels(ctx, advDeploy)
	if err == nil {
		if !metav1.IsControlledBy(svc, advDeploy) {
			err = controllerutil.SetControllerReference(advDeploy, svc, r.Mgr.GetScheme())
			if err == nil {
				err := r.Client.Update(ctx, svc)
				if err != nil {
					klog.Errorf("name: %s Update svc err: %#v", advDeploy.Name, err)
				}
			}
		}
	}
	var statefulSets []*appsv1.StatefulSet
	if len(deploys) == 0 {
		statefulSets, err = r.GetStatefulSetByLabels(ctx, advDeploy)
		if err != nil {
			return nil, errors.Wrapf(err, "Find all relative statefulSet with application name [%s] has an error", advDeploy.Name)
		}
	}

	unUseObject := make([]Object, 0)
	status := &workloadv1beta1.AdvDeploymentAggrStatus{}
	isGenerationEqual := true
	var updatedReplicas int32 = 0
	for _, deploy := range deploys {
		if !metav1.IsControlledBy(deploy, advDeploy) {
			err = controllerutil.SetControllerReference(advDeploy, deploy, r.Mgr.GetScheme())
			if err == nil {
				err := r.Client.Update(ctx, deploy)
				if err != nil {
					klog.Errorf("name: %s Update deploy: %s err: %#v", advDeploy.Name, deploy.Name, err)
				}
			}
		}

		if IsUnUseObject(DeploymentKind, deploy, ownerRes) {
			unUseObject = append(unUseObject, deploy)
			continue
		}

		podSetStatus := &workloadv1beta1.PodSetStatusInfo{}
		podSetStatus.Name = deploy.Name
		podSetStatus.Version = utils.FillImageVersion(advDeploy.Name, &deploy.Spec.Template.Spec)
		podSetStatus.Available = deploy.Status.AvailableReplicas
		podSetStatus.Desired = *deploy.Spec.Replicas
		podSetStatus.UnAvailable = deploy.Status.UnavailableReplicas
		podSetStatus.Update = &deploy.Status.UpdatedReplicas
		podSetStatus.Current = &deploy.Status.Replicas
		podSetStatus.Ready = &deploy.Status.ReadyReplicas

		status.Available += podSetStatus.Available
		status.Desired += podSetStatus.Desired
		status.UnAvailable += podSetStatus.UnAvailable
		updatedReplicas += deploy.Status.UpdatedReplicas

		status.PodSets = append(status.PodSets, podSetStatus)
		if deploy.Status.ObservedGeneration != deploy.ObjectMeta.Generation {
			isGenerationEqual = false
		}
	}

	for _, set := range statefulSets {
		if !metav1.IsControlledBy(set, advDeploy) {
			err = controllerutil.SetControllerReference(advDeploy, set, r.Mgr.GetScheme())
			if err == nil {
				err := r.Client.Update(ctx, set)
				if err != nil {
					klog.Errorf("name: %s Update set: %s err: %#v", advDeploy.Name, set.Name, err)
				}
			}
		}

		if IsUnUseObject(StatefulSetKind, set, ownerRes) {
			unUseObject = append(unUseObject, set)
			continue
		}

		podSetStatus := &workloadv1beta1.PodSetStatusInfo{}
		podSetStatus.Name = set.Name
		podSetStatus.Version = utils.FillImageVersion(advDeploy.Name, &set.Spec.Template.Spec)
		podSetStatus.Available = set.Status.ReadyReplicas
		podSetStatus.Desired = *set.Spec.Replicas
		podSetStatus.Update = &set.Status.UpdatedReplicas
		podSetStatus.Current = &set.Status.Replicas
		podSetStatus.Ready = &set.Status.ReadyReplicas

		status.Available += podSetStatus.Available
		status.Desired += podSetStatus.Desired
		updatedReplicas += set.Status.UpdatedReplicas

		status.PodSets = append(status.PodSets, podSetStatus)

		if set.Status.ObservedGeneration != set.ObjectMeta.Generation {
			isGenerationEqual = false
		}
	}

	sort.Slice(status.PodSets, func(i, j int) bool {
		return status.PodSets[i].Name < status.PodSets[j].Name
	})

	status.Version = utils.FillDuplicatedVersion(status.PodSets)

	if status.Desired == status.Available && status.UnAvailable == 0 && isGenerationEqual && status.Desired == updatedReplicas {
		status.Status = workloadv1beta1.AppStatusRuning
	} else {
		status.Status = workloadv1beta1.AppStatusInstalling
	}

	if status.Desired <= status.Available {
		for _, unobj := range unUseObject {
			klog.Infof("start delete unuse obj: %s/%s", unobj.GetNamespace(), unobj.GetName())
			err := r.Client.Delete(ctx, unobj)
			if err != nil {
				klog.Errorf("unuse obj[%s/%s] delete err:%v", unobj.GetNamespace(), unobj.GetName(), err)
			} else {
				klog.Infof("unuse obj[%s/%s] delete successfully", unobj.GetNamespace(), unobj.GetName())
			}
		}
	}

	status.OwnerResource = ownerRes
	return status, nil
}

//updateStatus Update the calculated status into CRD's status so that the controller which is watching for it can be noticed
func (r *AdvDeploymentReconciler) updateStatus(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment, recalStatus *workloadv1beta1.AdvDeploymentAggrStatus) error {
	obj := &workloadv1beta1.AdvDeployment{}

	nsName := types.NamespacedName{
		Name:      advDeploy.Name,
		Namespace: advDeploy.Namespace,
	}
	err := r.Client.Get(ctx, nsName, obj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.V(3).Infof("Can not find any advDeploy with name [%s], don't care about it.", nsName)
			return err
		}

		return err
	}

	if obj.Status.ObservedGeneration == obj.ObjectMeta.Generation && equality.Semantic.DeepEqual(&obj.Status.AggrStatus, recalStatus) {
		klog.V(4).Infof("advDeploy[%s]'s status is equal", advDeploy.Name)
		return nil
	}

	var updateErr error
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		now := metav1.Now()
		obj.Status.LastUpdateTime = &now
		recalStatus.DeepCopyInto(&obj.Status.AggrStatus)
		// It is very useful for controller that support this field
		// without this, you might trigger a sync as a result of updating your own status.
		obj.Status.ObservedGeneration = obj.ObjectMeta.Generation

		if r.Opt.OldCluster {
			updateErr = r.Client.Update(ctx, obj)
		} else {
			updateErr = r.Client.Status().Update(ctx, obj)
		}

		if updateErr == nil {
			klog.V(3).Infof("Updating the status of advDeploy[%s] successfully", advDeploy.Name)
			return nil
		}

		klog.Warningf("Update advDeploy[%s] status err: %v", advDeploy.Name, updateErr)

		// Get the advdeploy again when updating is failed.
		getErr := r.Client.Get(ctx, nsName, obj)
		if getErr != nil {
			utilruntime.HandleError(fmt.Errorf("getting updated Status advDeploy: [%s/%s] err: %v", advDeploy.Namespace, advDeploy.Name, err))
		}
		return updateErr
	})
	return err
}
