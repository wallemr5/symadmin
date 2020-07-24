package advdeployment

import (
	"context"
	"strconv"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/common"
	"gitlab.dmall.com/arch/sym-admin/pkg/helm/object"
	pkgLabels "gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources"
	"k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const targetAverageUtilization = "AverageUtilization"
const targetAverageValue = "AverageValue"

func BuildHorizontalPodAutoscaler(advDeploy *workloadv1beta1.AdvDeployment, object *object.K8sObject, apiVersion string, currentReplicas int32, mgr manager.Manager) (*v2beta2.HorizontalPodAutoscaler, resources.DesiredState) {
	hpa := &v2beta2.HorizontalPodAutoscaler{}
	objectkey := client.ObjectKey{Name: object.Name, Namespace: object.Namespace}
	ctx := context.TODO()
	if err := mgr.GetClient().Get(ctx, objectkey, hpa); err == nil {
		if hpa.Annotations[pkgLabels.WorkLoadAnnotationHpa] == advDeploy.Annotations[pkgLabels.WorkLoadAnnotationHpa] &&
			hpa.Annotations[pkgLabels.WorkLoadAnnotationHpaMetrics] == advDeploy.Annotations[pkgLabels.WorkLoadAnnotationHpaMetrics] {
			return nil, resources.DesiredStatePresent
		}
	}
	var defautlMetricValue int32
	defautlMetricValue = 70
	hpaLabels := make(map[string]string)
	hpaLabels["app"] = advDeploy.Name
	hpaLabels["app.kubernetes.io/instance"] = object.Name
	klog.Info("starting building hpa")
	hpa = &v2beta2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v2beta2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      object.Name,
			Namespace: object.Namespace,
			Labels:    hpaLabels,
		},
		Spec: v2beta2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: v2beta2.CrossVersionObjectReference{
				APIVersion: apiVersion,
				Kind:       object.Kind,
				Name:       object.Name,
			},
		},
	}

	hpaspec := common.GetHpaSpecObj(advDeploy.Annotations)
	if hpaspec == nil || !hpaspec.Enable {
		klog.Infof("not found hapspec annotations or hpa disable")
		return hpa, resources.DesiredStateAbsent
	}
	hpa.Spec.MaxReplicas = currentReplicas * 2
	hpa.Spec.MinReplicas = &currentReplicas
	metrics := parseMetrics(advDeploy.Annotations, object.Name)
	klog.Info("number of metrics: ", len(metrics))

	if len(metrics) == 0 {
		klog.Infof("create default metrics value")
		defaultCpuMetric := &v2beta2.MetricSpec{
			Type: v2beta2.ResourceMetricSourceType,
			Resource: &v2beta2.ResourceMetricSource{
				Name: "cpu",
				Target: v2beta2.MetricTarget{
					Type:               v2beta2.UtilizationMetricType,
					AverageUtilization: &defautlMetricValue,
				},
			},
		}
		defaultMemMetric := &v2beta2.MetricSpec{
			Type: v2beta2.ResourceMetricSourceType,
			Resource: &v2beta2.ResourceMetricSource{
				Name: "memory",
				Target: v2beta2.MetricTarget{
					Type:               v2beta2.UtilizationMetricType,
					AverageUtilization: &defautlMetricValue,
				},
			},
		}
		metrics = append(metrics, *defaultMemMetric, *defaultCpuMetric)
	}

	hpa.Spec.Metrics = metrics
	hpa.SetAnnotations(advDeploy.Annotations)
	controllerutil.SetControllerReference(advDeploy, hpa, mgr.GetScheme())
	return hpa, resources.DesiredStatePresent
}

func parseMetrics(annotations map[string]string, objectName string) []v2beta2.MetricSpec {

	metrics := make([]v2beta2.MetricSpec, 0, 4)
	metricSlice := common.GetHpaMetricObj(annotations)

	if len(metricSlice) == 0 {
		klog.Info("Annotations don't have metric")
		return metrics
	}

	for _, m := range metricSlice {
		var metric *v2beta2.MetricSpec
		switch m.ResourceName {
		case "cpu":
			metric = createResourceMetric(v1.ResourceCPU, m.MetricType, m.MetricValue, objectName)
		case "memory":
			metric = createResourceMetric(v1.ResourceMemory, m.MetricType, m.MetricValue, objectName)
		}
		if metric != nil {
			metrics = append(metrics, *metric)
		}

	}
	return metrics
}

func createResourceMetric(resourceName v1.ResourceName, metricType string, metricVaule string, deploymentName string) *v2beta2.MetricSpec {

	if len(metricType) == 0 {
		klog.Errorf("Invalid resource metric value format for deployment %v is missing", deploymentName)
		return nil
	}
	if len(metricVaule) == 0 {
		klog.Errorf("Invalid resource metric value  for deployment %v is missing", deploymentName)
		return nil
	}

	switch metricType {
	case targetAverageUtilization:
		int64Value, err := strconv.ParseInt(metricVaule, 10, 32)
		if err != nil {
			klog.Errorf("Invalid resource metric annotation: %v value for deployment %v is invalid: %v", metricVaule, deploymentName, err.Error())
			return nil
		}
		targetValue := int32(int64Value)
		if targetValue <= 0 || targetValue > 100 {
			klog.Errorf("Invalid resource metric annotation: %v value for deployment %v should be a percentage value between [1,99]", metricType, deploymentName)
			return nil
		}
		if targetValue <= 0 || targetValue > 100 {
			klog.Errorf("Invalid resource metric value for deployment %v should be a percentage value between [1,99]", deploymentName)
			return nil
		}

		if targetValue > 0 {
			return &v2beta2.MetricSpec{
				Type: v2beta2.ResourceMetricSourceType,
				Resource: &v2beta2.ResourceMetricSource{
					Name: resourceName,
					Target: v2beta2.MetricTarget{
						Type:               v2beta2.UtilizationMetricType,
						AverageUtilization: &targetValue,
					},
				},
			}
		}

	case targetAverageValue:
		targetValue, err := resource.ParseQuantity(metricVaule)
		if err != nil {
			klog.Errorf("Invalid resource metric value for deployment %v is invalid: %v", deploymentName, err.Error())
			return nil
		} else {
			return &v2beta2.MetricSpec{
				Type: v2beta2.ResourceMetricSourceType,
				Resource: &v2beta2.ResourceMetricSource{
					Name: resourceName,
					Target: v2beta2.MetricTarget{
						Type:         v2beta2.AverageValueMetricType,
						AverageValue: &targetValue,
					},
				},
			}
		}
	default:
		klog.Warningf("Invalid resource metric metricType: %v for deployment %v", metricType, deploymentName)
	}

	return nil
}
