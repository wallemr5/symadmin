package advdeployment

import (
	"context"
	"strconv"

	"emperror.dev/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/common"
	"gitlab.dmall.com/arch/sym-admin/pkg/helm/object"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources"
	"k8s.io/api/autoscaling/v2beta2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"

	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const targetAverageUtilization = "AverageUtilization"
const targetAverageValue = "AverageValue"

func GetDefautlCpuMetricValue() *int32 {
	var defautlMetricValue int32
	defautlMetricValue = 70

	return &defautlMetricValue
}

func GetDefautlMemMetricValue() *int32 {
	var defautlMetricValue int32
	defautlMetricValue = 70

	return &defautlMetricValue
}

func isHpaChanged(new, old *v2beta2.HorizontalPodAutoscaler) bool {
	if utils.IsObjectMetaChange(new, old) {
		return true
	}

	if !equality.Semantic.DeepEqual(new.Spec, old.Spec) {
		return true
	}
	return false
}

func applyHpa(cli client.Client, hpa *v2beta2.HorizontalPodAutoscaler) error {
	key := types.NamespacedName{
		Name:      hpa.Name,
		Namespace: hpa.Namespace,
	}
	name := key.String()
	ctx := context.TODO()
	current := &v2beta2.HorizontalPodAutoscaler{}
	err := cli.Get(ctx, key, current)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = cli.Create(ctx, hpa)
			if err != nil {
				klog.Errorf("%s create hpa failed, err: %+v", name, err)
				return errors.Wrapf(err, "%s create hpa failed", name)
			}
			klog.V(4).Infof("%s create hpa successfully", name)
			return nil
		}

		klog.Errorf("%s get hpa failed, err: %+v", name, err)
		return err
	}

	if isHpaChanged(hpa, current) {
		metaAccessor := meta.NewAccessor()
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			currentResourceVersion, err := metaAccessor.ResourceVersion(current)
			if err != nil {
				klog.Errorf("name: %s metaAccessor err: %+v", name, err)
				return err
			}

			metaAccessor.SetResourceVersion(hpa, currentResourceVersion)
			updateErr := cli.Update(ctx, hpa)
			if updateErr == nil {
				klog.V(5).Infof("%s update hpa successfully", name)
				return nil
			}

			getErr := cli.Get(ctx, key, current)
			if getErr != nil {
				klog.Errorf("%s get hpa failed, err: %+v", name, getErr)
			}
			return updateErr
		})

		if err != nil {
			klog.Warningf("%s update hpa, err: %+v", name, err)
			return err
		}

		return nil
	}

	return nil
}

func ApplyHorizontalPodAutoscaler(mgr manager.Manager, advDeploy *workloadv1beta1.AdvDeployment, object *object.K8sObject, apiVersion string, currentReplicas int32) error {
	isEnable := common.GetHpaSpecEnable(advDeploy.Annotations)
	if !isEnable || currentReplicas == 0 {
		klog.V(5).Infof("hpa not enable or obj name: %s replicas is zero", object.Name)
		hpa := &v2beta2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      object.Name,
				Namespace: object.Namespace,
			},
		}
		_, _ = resources.Reconcile(context.TODO(), mgr.GetClient(), hpa, resources.Option{DesiredState: resources.DesiredStateAbsent})
		return nil
	}

	metrics := parseMetrics(advDeploy.Annotations, object.Name)
	klog.Info("number of metrics: ", len(metrics))

	if len(metrics) == 0 {
		klog.V(5).Infof("create default metrics value")
		defaultCpuMetric := &v2beta2.MetricSpec{
			Type: v2beta2.ResourceMetricSourceType,
			Resource: &v2beta2.ResourceMetricSource{
				Name: "cpu",
				Target: v2beta2.MetricTarget{
					Type:               v2beta2.UtilizationMetricType,
					AverageUtilization: GetDefautlCpuMetricValue(),
				},
			},
		}
		defaultMemMetric := &v2beta2.MetricSpec{
			Type: v2beta2.ResourceMetricSourceType,
			Resource: &v2beta2.ResourceMetricSource{
				Name: "memory",
				Target: v2beta2.MetricTarget{
					Type:               v2beta2.UtilizationMetricType,
					AverageUtilization: GetDefautlMemMetricValue(),
				},
			},
		}
		metrics = append(metrics, *defaultMemMetric, *defaultCpuMetric)
	}

	hpa := &v2beta2.HorizontalPodAutoscaler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "HorizontalPodAutoscaler",
			APIVersion: "autoscaling/v2beta2",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      object.Name,
			Namespace: object.Namespace,
			Labels: map[string]string{
				"app":                        advDeploy.Name,
				"app.kubernetes.io/instance": object.Name,
			},
		},
		Spec: v2beta2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: v2beta2.CrossVersionObjectReference{
				APIVersion: apiVersion,
				Kind:       object.Kind,
				Name:       object.Name,
			},
			Metrics:     metrics,
			MinReplicas: &currentReplicas,
			MaxReplicas: currentReplicas * 2,
		},
	}

	controllerutil.SetControllerReference(advDeploy, hpa, mgr.GetScheme())
	klog.V(4).Infof("starting apply hpa name: %s minReplicas: %d maxReplicas: %d",
		hpa.Name, *hpa.Spec.MinReplicas, hpa.Spec.MaxReplicas)
	err := applyHpa(mgr.GetClient(), hpa)
	if err != nil {
		klog.Errorf("apply hpa name: %s, err: %+v", hpa.Name, err)
	}
	return nil
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
