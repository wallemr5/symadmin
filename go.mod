module gitlab.dmall.com/arch/sym-admin

go 1.13

require (
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/emicklei/go-restful v2.10.0+incompatible // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.3 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/goph/emperror v0.17.2
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/opencontainers/go-digest v1.0.0-rc1 // indirect
	github.com/openkruise/kruise v0.2.0
	github.com/pkg/errors v0.8.1
	k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/klog v0.4.0
	k8s.io/kubernetes v1.14.6
	sigs.k8s.io/controller-runtime v0.2.2
)

replace (
	// Kubernetes 1.14.6
	k8s.io/kubernetes => k8s.io/kubernetes v1.14.6
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.2.2
)
