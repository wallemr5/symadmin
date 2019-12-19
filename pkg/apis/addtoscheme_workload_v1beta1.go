package apis

import (
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	extensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, workloadv1beta1.SchemeBuilder.AddToScheme)
	AddToSchemes = append(AddToSchemes, extensionsv1beta1.AddToScheme)
}
