/*
Copyright 2019 The dks authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
