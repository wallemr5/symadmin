package advdeployment

import (
	"context"
	"fmt"
	"sort"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/helm/object"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"emperror.dev/errors"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/common"
	pkgLabels "gitlab.dmall.com/arch/sym-admin/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	StatefulSetKind = "StatefulSet"
	DeploymentKind  = "Deployment"
	ServiceKind     = "Service"
)

type Object interface {
	metav1.Object
	runtime.Object
}

func GetFormattedName(kind string, object Object) string {
	return fmt.Sprintf("%s:%s/%s", kind, object.GetNamespace(), object.GetName())
}

func FillWorkloadSelectorLabel(labels map[string]string) map[string]string {
	lb := make(map[string]string)

	if va, ok := labels[pkgLabels.ObserveMustLabelAppName]; !ok {
		lb[pkgLabels.ObserveMustLabelAppName] = va
	}

	if vg, ok := labels[pkgLabels.ObserveMustLabelGroupName]; !ok {
		lb[pkgLabels.ObserveMustLabelGroupName] = vg
		lb[pkgLabels.ObserveMustLabelVersion] = vg
	}

	return lb
}

func ConvertToLabel(labels map[string]string) {
	if vg, ok := labels[pkgLabels.ObserveMustLabelGroupName]; !ok {
		labels[pkgLabels.ObserveMustLabelVersion] = vg
	}
}

func ConvertToSvc(mgr manager.Manager, obj *unstructured.Unstructured) (*corev1.Service, error) {
	var svc corev1.Service
	err := mgr.GetScheme().Convert(obj, &svc, nil)
	if err != nil {
		return nil, err
	}

	return &svc, nil
}

func ConvertToDeployment(mgr manager.Manager, obj *unstructured.Unstructured) (*appsv1.Deployment, error) {
	var deploy appsv1.Deployment
	err := mgr.GetScheme().Convert(obj, &deploy, nil)
	if err != nil {
		return nil, err
	}

	// ConvertToLabel(deploy.Labels)
	// ConvertToLabel(deploy.Spec.Template.Labels)

	if deploy.Spec.RevisionHistoryLimit == nil {
		deploy.Spec.RevisionHistoryLimit = utils.IntPointer(10)
	}

	if deploy.Spec.ProgressDeadlineSeconds == nil {
		deploy.Spec.ProgressDeadlineSeconds = utils.IntPointer(600)
	}

	if deploy.Spec.Selector == nil {
		deploy.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: deploy.Spec.Template.Labels,
		}
	}

	// if deploy.Spec.Template.Spec.Affinity == nil {
	// 	deploy.Spec.Template.Spec.Affinity = GetAffinity(advDeploy)
	// }

	return &deploy, nil
}

func ConvertToStatefulSet(mgr manager.Manager, obj *unstructured.Unstructured) (*appsv1.StatefulSet, error) {
	var sta appsv1.StatefulSet
	err := mgr.GetScheme().Convert(obj, &sta, nil)
	if err != nil {
		return nil, err
	}

	// ConvertToLabel(sta.Labels)
	// ConvertToLabel(sta.Spec.Template.Labels)
	if sta.Spec.RevisionHistoryLimit == nil {
		sta.Spec.RevisionHistoryLimit = utils.IntPointer(10)
	}

	if sta.Spec.Selector == nil {
		sta.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: FillWorkloadSelectorLabel(sta.Spec.Template.Labels),
		}
	}

	// if sta.Spec.Template.Spec.Affinity == nil {
	// 	sta.Spec.Template.Spec.Affinity = GetAffinity(advDeploy)
	// }
	return &sta, nil
}

func (r *AdvDeploymentReconciler) ApplyResources(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment) ([]string, int, error) {
	var isChanged int
	var objects object.K8sObjects

	for _, podSet := range advDeploy.Spec.Topology.PodSets {
		_, _, specRawChart := getChartInfo(podSet, advDeploy)
		obj, err := object.RenderTemplate(specRawChart, podSet.Name, advDeploy.Namespace, podSet.RawValues)
		if err != nil {
			klog.Errorf("Template podSet Name: %s err: %v", podSet.Name, err)
			return nil, isChanged, errors.Wrapf(err, "podSet: %s parse k8s object", podSet.Name)
		}
		objects = append(objects, obj...)
	}

	ownerRes := make([]string, 0)
	isHpaEnable := common.GetHpaSpecEnable(advDeploy.Annotations)
	for _, obj := range objects {
		yml := obj.YAMLDebugString()
		klog.V(5).Infof("kind: %s, Name: %s/%s, obj:\n%s", obj.Kind, obj.Namespace, obj.Name, yml)
		switch obj.Kind {
		case ServiceKind:
			svc, err := ConvertToSvc(r.Mgr, obj.UnstructuredObject())
			if err != nil {
				klog.Errorf("failed convert kind: %s, Name: %s/%s, err: %v", obj.Kind, obj.Namespace, obj.Name, err)
				return nil, isChanged, errors.Wrapf(err, "failed convert kind: %s Name: %s/%s, obj:\n%s",
					obj.Kind, obj.Namespace, obj.Name, yml)
			}

			ownerRes = append(ownerRes, GetFormattedName(ServiceKind, svc))
			change, err := resources.Reconcile(ctx, r.Client, svc, resources.Option{IsRecreate: r.Opt.Debug})
			if err != nil {
				klog.Errorf("svc name: %s err: %v", svc.Name, err)
				return nil, isChanged, errors.Wrapf(err, "reconcile advDeploy: %s svc: %s", advDeploy.Name, obj.Name)
			}
			if change > 0 {
				isChanged++
			}
		case DeploymentKind:
			deploy, err := ConvertToDeployment(r.Mgr, obj.UnstructuredObject())
			if err != nil {
				klog.Errorf("failed convert kind: %s, Name: %s/%s, err: %v", obj.Kind, obj.Namespace, obj.Name, err)
				return nil, isChanged, errors.Wrapf(err, "failed convert kind: %s Name: %s/%s, obj:\n%s",
					obj.Kind, obj.Namespace, obj.Name, yml)
			}
			ownerRes = append(ownerRes, GetFormattedName(DeploymentKind, deploy))
			change, err := resources.Reconcile(ctx, r.Client, deploy, resources.Option{IsRecreate: r.Opt.Debug, IsIgnoreReplicas: isHpaEnable})
			if err != nil {
				klog.Errorf("deployment name: %s err: %v", deploy.Name, err)
				return nil, isChanged, errors.Wrapf(err, "reconcile advDeploy: %s deployment: %s", advDeploy.Name, obj.Name)
			}

			_ = ApplyHorizontalPodAutoscaler(r.Mgr, advDeploy, obj, "apps/v1", *deploy.Spec.Replicas)
			if change > 0 {
				isChanged++
			}
		case StatefulSetKind:
			sta, err := ConvertToStatefulSet(r.Mgr, obj.UnstructuredObject())
			if err != nil {
				klog.Errorf("failed convert kind: %s, Name: %s/%s, err: %v", obj.Kind, obj.Namespace, obj.Name, err)
				return nil, isChanged, errors.Wrapf(err, "failed convert kind: %s Name: %s/%s, obj:\n%s",
					obj.Kind, obj.Namespace, obj.Name, yml)
			}
			ownerRes = append(ownerRes, GetFormattedName(StatefulSetKind, sta))
			change, err := resources.Reconcile(ctx, r.Client, sta, resources.Option{IsRecreate: r.Opt.Debug, IsIgnoreReplicas: isHpaEnable})
			if err != nil {
				klog.Errorf("statefulset name: %s err: %v", sta.Name, err)
				return nil, isChanged, errors.Wrapf(err, "reconcile advDeploy: %s statefulset: %s", advDeploy.Name, obj.Name)
			}

			_ = ApplyHorizontalPodAutoscaler(r.Mgr, advDeploy, obj, "apps/v1", *sta.Spec.Replicas)
			if change > 0 {
				isChanged++
			}
		default:
			return nil, isChanged, fmt.Errorf("unknown kind: %s Name: %s/%s ", obj.Kind, obj.Namespace, obj.Name)
		}
	}

	return ownerRes, isChanged, nil
}

func GetAffinity(advDeploy *workloadv1beta1.AdvDeployment) *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 1,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								pkgLabels.ObserveMustLabelAppName: advDeploy.Name,
							},
						},
						Namespaces:  []string{advDeploy.Namespace},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}
}

// GetStatefulSetByLabels Finding all statefuls with a label
func (r *AdvDeploymentReconciler) GetStatefulSetByLabels(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment) ([]*appsv1.StatefulSet, error) {
	opts := &client.ListOptions{
		Namespace:     advDeploy.Namespace,
		LabelSelector: labels.Set{"app": advDeploy.Name}.AsSelector(),
	}

	staSets := &appsv1.StatefulSetList{}
	err := r.Client.List(ctx, staSets, opts)
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

	opts := &client.ListOptions{
		Namespace:     advDeploy.Namespace,
		LabelSelector: labels.Set{"app": advDeploy.Name}.AsSelector(),
	}

	deployLists := &appsv1.DeploymentList{}
	err := r.Client.List(ctx, deployLists, opts)
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
	opts := &client.ListOptions{
		Namespace:     advDeploy.Namespace,
		LabelSelector: labels.Set{"app": advDeploy.Name + "-svc"}.AsSelector(),
	}

	svcLists := &corev1.ServiceList{}
	err := r.Client.List(ctx, svcLists, opts)
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
func (r *AdvDeploymentReconciler) RecalculateStatus(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment, ownerRes []string) (*workloadv1beta1.AdvDeploymentAggrStatus, bool, error) {
	deploys, err := r.GetDeployListByByLabels(ctx, advDeploy)
	if err != nil {
		return nil, false, errors.Wrapf(err, "Find all relative deployments with application name [%s] has an error", advDeploy.Name)
	}

	svc, err := r.GetServiceByByLabels(ctx, advDeploy)
	if err == nil {
		if !metav1.IsControlledBy(svc, advDeploy) {
			err = controllerutil.SetControllerReference(advDeploy, svc, r.Mgr.GetScheme())
			if err == nil {
				err := r.Client.Update(ctx, svc)
				if err != nil {
					klog.Errorf("name [%s] update svc Owner err: %#v", advDeploy.Name, err)
				}
			}
		}
	}
	var statefulSets []*appsv1.StatefulSet
	if len(deploys) == 0 {
		statefulSets, err = r.GetStatefulSetByLabels(ctx, advDeploy)
		if err != nil {
			return nil, false, errors.Wrapf(err, "name [%s] get statefulset by label", advDeploy.Name)
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
					klog.Errorf("name [%s] update deploy [%s] Owner err: %#v", advDeploy.Name, deploy.Name, err)
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
	return status, isGenerationEqual, nil
}

// updateStatus Update the calculated status into CRD's status so that the controller which is watching for it can be noticed
func (r *AdvDeploymentReconciler) updateStatus(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment, recalStatus *workloadv1beta1.AdvDeploymentAggrStatus, isGenerationEqual bool) error {
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
		if isGenerationEqual {
			obj.Status.ObservedGeneration = obj.ObjectMeta.Generation
		} else {
			obj.Status.ObservedGeneration = obj.ObjectMeta.Generation - 1
		}

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
