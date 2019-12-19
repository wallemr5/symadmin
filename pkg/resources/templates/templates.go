package templates

import (
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	// appsv1 "k8s.io/api/apps/v1"
	// corev1 "k8s.io/api/core/v1"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ObjectMeta(name string, labels map[string]string, config *workloadv1beta1.AdvDeployment) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: config.Namespace,
		Labels:    labels,
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion:         config.APIVersion,
				Kind:               config.Kind,
				Name:               config.Name,
				UID:                config.UID,
				Controller:         utils.BoolPointer(true),
				BlockOwnerDeletion: utils.BoolPointer(true),
			},
		},
	}
}

func ObjectMetaWithAnnotations(name string, labels map[string]string, annotations map[string]string, config *workloadv1beta1.AdvDeployment) metav1.ObjectMeta {
	o := ObjectMeta(name, labels, config)
	o.Annotations = annotations
	return o
}

func ObjectMetaClusterScope(name string, labels map[string]string, config *workloadv1beta1.AdvDeployment) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:   name,
		Labels: labels,
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion:         config.APIVersion,
				Kind:               config.Kind,
				Name:               config.Name,
				UID:                config.UID,
				Controller:         utils.BoolPointer(true),
				BlockOwnerDeletion: utils.BoolPointer(true),
			},
		},
	}
}

func ControlPlaneAuthPolicy(enabled bool) string {
	if enabled {
		return "MUTUAL_TLS"
	}
	return "NONE"
}
