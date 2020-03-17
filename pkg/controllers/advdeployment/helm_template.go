package advdeployment

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/helm/object"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources/patch"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/helm/pkg/chartutil"
	helmenv "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/timeconv"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type DesiredState string

const (
	DesiredStatePresent DesiredState = "present"
	DesiredStateAbsent  DesiredState = "absent"

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

func TemplateK8sObject(rlsName, chartName, chartVersion string, chartPackage []byte, env *helmenv.EnvSettings, namespace string, overrideValue string) (object.K8sObjects, error) {
	chrt, err := helmv2.GetRequestedChart(rlsName, chartName, chartVersion, chartPackage, env)
	if err != nil {
		return nil, fmt.Errorf("loading chart has an error: %v", err)
	}

	renderOpts := renderutil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      rlsName,
			IsInstall: true,
			IsUpgrade: false,
			Time:      timeconv.Now(),
			Namespace: namespace,
		},
		KubeVersion: "",
	}

	chrtConfig := &chart.Config{
		Raw:    overrideValue,
		Values: map[string]*chart.Value{},
	}

	renderedTemplates, err := renderutil.Render(chrt, chrtConfig, renderOpts)
	if err != nil {
		klog.Errorf("Render err:%#v", err)
		return nil, err
	}

	var objects []*object.K8sObject
	for name, yaml := range renderedTemplates {
		yaml = object.RemoveNonYAMLLines(yaml)
		if yaml == "" {
			continue
		}
		klog.V(4).Infof("start ation name: %s ... \n %s", name, yaml)
		o, err := object.ParseYAMLToK8sObject([]byte(yaml))
		if err != nil {
			klog.Errorf("Failed to parse YAML to a k8s object: %v", err.Error())
			continue
		}

		objects = append(objects, o)
	}

	return objects, nil
}

func FillWorkloadSelectorLabel(labels map[string]string) map[string]string {
	lb := make(map[string]string)

	if va, ok := labels[utils.ObserveMustLabelAppName]; !ok {
		lb[utils.ObserveMustLabelAppName] = va
	}

	if vg, ok := labels[utils.ObserveMustLabelGroupName]; !ok {
		lb[utils.ObserveMustLabelGroupName] = vg
		lb[utils.ObserveMustLabelVersion] = vg
	}

	return lb
}

func ConvertToLabel(labels map[string]string) {
	if vg, ok := labels[utils.ObserveMustLabelGroupName]; !ok {
		labels[utils.ObserveMustLabelVersion] = vg
	}
}

func ConvertToSvc(mgr manager.Manager, advDeploy *workloadv1beta1.AdvDeployment, obj *unstructured.Unstructured) (*corev1.Service, bool) {
	var svc corev1.Service
	err := mgr.GetScheme().Convert(obj, &svc, nil)
	if err != nil {
		klog.Warningf("convert svc name:%s err: %#v", advDeploy.Name, err)
		return nil, false
	}

	return &svc, true
}

func ConvertToDeployment(mgr manager.Manager, advDeploy *workloadv1beta1.AdvDeployment, obj *unstructured.Unstructured) (*appsv1.Deployment, bool) {
	var deploy appsv1.Deployment
	err := mgr.GetScheme().Convert(obj, &deploy, nil)
	if err != nil {
		klog.Warningf("convert deploy name:%s err: %#v", advDeploy.Name, err)
		return nil, false
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

	return &deploy, true
}

func ConvertToStatefulSet(mgr manager.Manager, advDeploy *workloadv1beta1.AdvDeployment, obj *unstructured.Unstructured) (*appsv1.StatefulSet, bool) {
	var sta appsv1.StatefulSet
	err := mgr.GetScheme().Convert(obj, &sta, nil)
	if err != nil {
		klog.Warningf("convert StatefulSet name:%s err: %#v", advDeploy.Name, err)
		return nil, false
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
	return &sta, true
}

func (r *AdvDeploymentReconciler) ApplyResources(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment) ([]string, error) {
	var err error
	var objects object.K8sObjects

	for _, podSet := range advDeploy.Spec.Topology.PodSets {
		specChartURLName, specChartURLVersion, specRawChart := getChartInfo(podSet, advDeploy)
		obj, err := TemplateK8sObject(podSet.Name, specChartURLName, specChartURLVersion, specRawChart,
			r.HelmEnv.Helmv2env, advDeploy.Namespace, podSet.RawValues)
		if err != nil {
			klog.Errorf("Template podSet Name: %s err: %v", podSet.Name, err)
			continue
		}
		objects = append(objects, obj...)
	}

	ownerRes := make([]string, 0)
	for _, obj := range objects {
		switch obj.Kind {
		case ServiceKind:
			svc, ok := ConvertToSvc(r.Mgr, advDeploy, obj.UnstructuredObject())
			if ok {
				ownerRes = append(ownerRes, GetFormattedName(ServiceKind, svc))
				err = Reconcile(ctx, r, svc, advDeploy, DesiredStatePresent)
				if err != nil {
					klog.Errorf("svc name: %s err: %v", svc.Name, err)
				}
			}
		case DeploymentKind:
			deploy, ok := ConvertToDeployment(r.Mgr, advDeploy, obj.UnstructuredObject())
			if ok {
				ownerRes = append(ownerRes, GetFormattedName(DeploymentKind, deploy))
				err = Reconcile(ctx, r, deploy, advDeploy, DesiredStatePresent)
				if err != nil {
					klog.Errorf("deploy name: %s err: %v", deploy.Name, err)
				}
			}
		case StatefulSetKind:
			sta, ok := ConvertToStatefulSet(r.Mgr, advDeploy, obj.UnstructuredObject())
			if ok {
				ownerRes = append(ownerRes, GetFormattedName(ServiceKind, sta))
				err = Reconcile(ctx, r, sta, advDeploy, DesiredStatePresent)
				if err != nil {
					klog.Errorf("sta name: %s err: %v", sta.Name, err)
				}
			}
		default:
		}
	}

	return ownerRes, nil
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
								utils.ObserveMustLabelAppName: advDeploy.Name,
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

func prepareResourceForUpdate(current, desired runtime.Object) {
	switch desired.(type) {
	case *corev1.Service:
		svc := desired.(*corev1.Service)
		svc.Spec.ClusterIP = current.(*corev1.Service).Spec.ClusterIP
	}
}

func Reconcile(ctx context.Context, r *AdvDeploymentReconciler, desired Object, owner Object, desiredState DesiredState) error {
	if desiredState == "" {
		desiredState = DesiredStatePresent
	}

	c := r.Client
	var current = desired.DeepCopyObject()
	// desiredType := reflect.TypeOf(desired)
	// var desiredCopy = desired.DeepCopyObject()
	key, err := client.ObjectKeyFromObject(current)
	if err != nil {
		return errors.Wrapf(err, "get key[%s]", key)
	}

	err = c.Get(ctx, key, current)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "getting resource failed key[%s]", key)
	}
	if apierrors.IsNotFound(err) {
		if desiredState == DesiredStatePresent {
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(desired); err != nil {
				klog.Errorf("Failed to set last applied annotation key[%s] err: %v", key, err)
			}
			if err := c.Create(ctx, desired); err != nil {
				return errors.Wrapf(err, "creating resource failed key[%s]", key)
			}
			klog.Infof("resource created")
		}
	} else {
		if desiredState == DesiredStatePresent {
			patchResult, err := patch.DefaultPatchMaker.Calculate(current, desired)
			if err != nil {
				klog.Errorf("could not match object key[%s] err: %v", key, err)
			} else if patchResult.IsEmpty() {
				klog.V(4).Infof("resource key[%s] unchanged is in sync", key)
				return nil
			} else {
				klog.V(2).Infof("resource key[%s] diffs patch: %s", key, string(patchResult.Patch))
			}

			// Need to set this before resourceversion is set, as it would constantly change otherwise
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(desired); err != nil {
				klog.Errorf("Failed to set last applied annotation key[%s] err: %v", key, err)
			}

			metaAccessor := meta.NewAccessor()
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				currentResourceVersion, err := metaAccessor.ResourceVersion(current)
				if err != nil {
					return err
				}

				metaAccessor.SetResourceVersion(desired, currentResourceVersion)
				prepareResourceForUpdate(current, desired)

				updateErr := r.Client.Update(ctx, desired)
				if updateErr == nil {
					klog.V(2).Infof("Updating resource key[%s] successfully", key)
					return nil
				}

				// Get the advdeploy again when updating is failed.
				getErr := r.Client.Get(ctx, key, current)
				if getErr != nil {
					return errors.Wrapf(err, "updated get resource key[%s] err: %v", key, err)
				}

				return updateErr
			})

			// if err := c.Update(ctx, desired); err != nil {
			// 	if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
			// 		klog.Infof("resource key:%s needs to be re-created err: %v", key, err)
			// 		err := c.Delete(ctx, current)
			// 		if err != nil {
			// 			return errors.Wrapf(err, "could not delete resource key:%s", key)
			// 		}
			// 		klog.Infof("resource key:%s deleted", key)
			// 		if err := c.Create(ctx, desiredCopy); err != nil {
			// 			return errors.Wrapf(err, "creating resource failed key:%s", key)
			// 		}
			// 		klog.Infof("resource key:%s created", key)
			// 		return nil
			// 	}
			// 	return errors.Wrapf(err, "updating resource key:%s failed", key)
			// }
			// klog.Infof("resource key:%s updated", key)
			return err
		} else if desiredState == DesiredStateAbsent {
			if err := c.Delete(ctx, current); err != nil {
				return errors.Wrapf(err, "deleting resource key[%s] failed", key)
			}
			klog.Infof("resource key[%s] deleted", key)
		}
	}
	return nil
}
