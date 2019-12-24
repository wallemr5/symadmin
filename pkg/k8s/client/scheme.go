package client

import (
	kruisev1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	//  monitorv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	//  networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	//  configv1alpha2 "istio.io/client-go/pkg/apis/config/v1alpha2
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = apiextensionsv1beta1.AddToScheme(scheme)
	_ = workloadv1beta1.AddToScheme(scheme)
	_ = kruisev1alpha1.AddToScheme(scheme)

	// _ = monitorv1.AddToScheme(scheme)
	// _ = networkingv1alpha3.AddToScheme(scheme)
	// _ = configv1alpha2.AddToScheme(scheme)
}

// GetScheme gets an initialized runtime.Scheme with k8s core added by default
func GetScheme() *runtime.Scheme {
	return scheme
}
