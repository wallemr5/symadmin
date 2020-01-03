package advdeployment

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
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
)

// fillImageVersion
func fillImageVersion(name string, podSpec *corev1.PodSpec) string {
	if podSpec == nil {
		return ""
	}

	for i := range podSpec.Containers {
		c := &podSpec.Containers[i]
		if c.Name == name {
			fullName := strings.Split(c.Image, ":")
			if len(fullName) > 1 {
				return fullName[1]
			}
		}
	}

	return ""
}

// fillDuplicatedVersion
func fillDuplicatedVersion(infos []workloadv1beta1.PodSetSatusInfo) string {
	found := make(map[string]bool)
	var foundSet []string
	for i := range infos {
		if infos[i].Version != "" {
			found[infos[i].Version] = true
		}
	}

	for k, _ := range found {
		foundSet = append(foundSet, k)
	}

	sort.Slice(foundSet, func(i, j int) bool {
		return foundSet[i] < foundSet[i]
	})

	return strings.Join(foundSet, "/")
}

func (r *AdvDeploymentReconciler) GetStatefulSetByLabels(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment, opts *client.ListOptions) ([]*appsv1.StatefulSet, error) {
	statefulSets := make([]*appsv1.StatefulSet, 0, 8)

	listOptions := &client.ListOptions{}
	listOptions.MatchingLabels(map[string]string{
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

	for i := range staSets.Items {
		statefulSets = append(statefulSets, &staSets.Items[i])
	}
	return statefulSets, nil
}

func (r *AdvDeploymentReconciler) GetDeployListByByLabels(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment, opts *client.ListOptions) ([]*appsv1.Deployment, error) {
	deploys := make([]*appsv1.Deployment, 0)

	deployLists := appsv1.DeploymentList{}
	err := r.Client.List(ctx, opts, &deployLists)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, err
		}

		klog.Errorf("failed to DeploymentList name:%s, err: %v", advDeploy.Name, err)
		return nil, err
	}

	for i := range deployLists.Items {
		deploys = append(deploys, &deployLists.Items[i])
	}

	return deploys, nil
}

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

		klog.Errorf("failed to ServiceList name:%s, err: %v", advDeploy.Name, err)
		return nil, err
	}

	if len(svcLists.Items) != 1 {
		return nil, fmt.Errorf("name: %s, have more or less svc len:%d", advDeploy.Name, len(svcLists.Items))
	} else {
		return &svcLists.Items[0], nil
	}
}

func (r *AdvDeploymentReconciler) RecalculateAppSetStatus(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment) (*workloadv1beta1.AdvDeploymentAggrStatus, error) {
	opts := &client.ListOptions{}
	opts.MatchingLabels(map[string]string{
		"app": advDeploy.Name,
	})

	deploys, err := r.GetDeployListByByLabels(ctx, advDeploy, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "app:%s get all deploy err", advDeploy.Name)
	}

	var statefulSets []*appsv1.StatefulSet
	if len(deploys) == 0 {
		statefulSets, err = r.GetStatefulSetByLabels(ctx, advDeploy, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "app:%s get all statts err", advDeploy.Name)
		}
	}

	status := &workloadv1beta1.AdvDeploymentAggrStatus{}
	for _, deploy := range deploys {
		podSetStatus := workloadv1beta1.PodSetSatusInfo{}
		podSetStatus.Name = deploy.Name
		podSetStatus.Version = fillImageVersion(advDeploy.Name, &deploy.Spec.Template.Spec)
		podSetStatus.Available = deploy.Status.AvailableReplicas
		podSetStatus.Desired = deploy.Status.Replicas

		status.Available += deploy.Status.AvailableReplicas
		status.Desired += deploy.Status.Replicas
		status.PodSets = append(status.PodSets, podSetStatus)
	}

	for _, set := range statefulSets {
		podSetStatus := workloadv1beta1.PodSetSatusInfo{}
		podSetStatus.Name = set.Name
		podSetStatus.Version = fillImageVersion(advDeploy.Name, &set.Spec.Template.Spec)
		podSetStatus.Available = set.Status.ReadyReplicas
		podSetStatus.Desired = set.Status.Replicas

		status.Available += set.Status.ReadyReplicas
		status.Desired += set.Status.Replicas

		status.PodSets = append(status.PodSets, podSetStatus)
	}

	sort.Slice(status.PodSets, func(i, j int) bool {
		return status.PodSets[i].Name < status.PodSets[i].Name
	})

	status.Version = fillDuplicatedVersion(status.PodSets)
	return status, nil
}

func (r *AdvDeploymentReconciler) updateAggrStatus(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment, aggrStatus *workloadv1beta1.AdvDeploymentAggrStatus) error {
	obj := &workloadv1beta1.AdvDeployment{}
	err := r.Client.Get(ctx, types.NamespacedName{
		Name:      advDeploy.Name,
		Namespace: advDeploy.Namespace,
	}, obj)

	if err != nil {
		if apierrors.IsNotFound(err) {
			return err
		}
		return err
	}

	if equality.Semantic.DeepEqual(&obj.Status.AggrStatus, aggrStatus) {
		return nil
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		time := metav1.Now()
		obj.Status.LastUpdateTime = &time
		aggrStatus.DeepCopyInto(&obj.Status.AggrStatus)
		updateErr := r.Client.Status().Update(ctx, obj)
		if updateErr == nil {
			klog.Infof("advDeploy name：%s Status updated successfully", advDeploy.Name)
			return nil
		}

		getErr := r.Client.Get(ctx, types.NamespacedName{
			Name:      advDeploy.Name,
			Namespace: advDeploy.Namespace,
		}, obj)

		if getErr != nil {
			utilruntime.HandleError(fmt.Errorf("getting updated Status advDeploy: [%s/%s] err: %v", advDeploy.Namespace, advDeploy.Name, err))
		}
		return updateErr
	})
	return err
}
