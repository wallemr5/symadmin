package advdeployment

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/helm/object"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/helm/pkg/chartutil"
	helmenv "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/timeconv"
	"k8s.io/klog"
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
			return nil, errors.Wrapf(err, "Resource name: %s Failed to parse YAML to a k8s object", name)
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
			return nil, fmt.Errorf("podSet: %s parse k8s object failed", podSet.Name)
		}
		objects = append(objects, obj...)
	}

	ownerRes := make([]string, 0)
	for _, obj := range objects {
		switch obj.Kind {
		case ServiceKind:
			svc, ok := ConvertToSvc(r.Mgr, advDeploy, obj.UnstructuredObject())
			if !ok {
				return nil, fmt.Errorf("Convert Failed kind: %s Name: %s/%s ", obj.Kind, obj.Namespace, obj.Name)
			}

			ownerRes = append(ownerRes, GetFormattedName(ServiceKind, svc))
			err = resources.Reconcile(ctx, r.Client, svc, resources.DesiredStatePresent, r.Opt.Debug)
			if err != nil {
				klog.Errorf("svc name: %s err: %v", svc.Name, err)
				return nil, fmt.Errorf("advDeploy: %s svc object: %s reconcile failed", advDeploy.Name, obj.Name)
			}
		case DeploymentKind:
			deploy, ok := ConvertToDeployment(r.Mgr, advDeploy, obj.UnstructuredObject())
			if !ok {
				return nil, fmt.Errorf("Convert Failed kind: %s Name: %s/%s ", obj.Kind, obj.Namespace, obj.Name)
			}
			ownerRes = append(ownerRes, GetFormattedName(DeploymentKind, deploy))
			err = resources.Reconcile(ctx, r.Client, deploy, resources.DesiredStatePresent, r.Opt.Debug)
			if err != nil {
				klog.Errorf("deployment name: %s err: %v", deploy.Name, err)
				return nil, fmt.Errorf("advDeploy: %s deployment object: %s reconcile failed", advDeploy.Name, obj.Name)
			}
		case StatefulSetKind:
			sta, ok := ConvertToStatefulSet(r.Mgr, advDeploy, obj.UnstructuredObject())
			if !ok {
				return nil, fmt.Errorf("Convert Failed kind: %s Name: %s/%s ", obj.Kind, obj.Namespace, obj.Name)
			}
			ownerRes = append(ownerRes, GetFormattedName(ServiceKind, sta))
			err = resources.Reconcile(ctx, r.Client, sta, resources.DesiredStatePresent, r.Opt.Debug)
			if err != nil {
				klog.Errorf("statefulset name: %s err: %v", sta.Name, err)
				return nil, fmt.Errorf("advDeploy: %s statefulset object: %s reconcile failed", advDeploy.Name, obj.Name)
			}
		default:
			return nil, fmt.Errorf("unknown kind: %s Name: %s/%s ", obj.Kind, obj.Namespace, obj.Name)
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
