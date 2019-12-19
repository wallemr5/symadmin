package templates

import (
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func DefaultDeployAnnotations() map[string]string {
	return map[string]string{}
}

func GetResourcesRequirementsOrDefault(requirements *corev1.ResourceRequirements, defaults *corev1.ResourceRequirements) corev1.ResourceRequirements {
	if requirements != nil {
		return *requirements
	}

	return *defaults
}

func DefaultRollingUpdateStrategy() appsv1.DeploymentStrategy {
	return appsv1.DeploymentStrategy{
		RollingUpdate: &appsv1.RollingUpdateDeployment{
			MaxSurge:       utils.IntstrPointer(1),
			MaxUnavailable: utils.IntstrPointer(0),
		},
	}
}

func DefaultRecreateStrategy() appsv1.DeploymentStrategy {
	return appsv1.DeploymentStrategy{
		Type: appsv1.RecreateDeploymentStrategyType,
	}
}

func AppEnv(config *workloadv1beta1.AdvDeployment) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.name",
				},
			},
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.namespace",
				},
			},
		},
		{
			Name: "INSTANCE_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "status.podIP",
				},
			},
		},
	}

	return envs
}
