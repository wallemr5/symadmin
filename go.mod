module gitlab.dmall.com/arch/sym-admin

go 1.13

require (
	github.com/DeanThompson/ginpprof v0.0.0-20190408063150-3be636683586
	github.com/gin-gonic/gin v1.5.0
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/goph/emperror v0.17.2
	github.com/openkruise/kruise v0.3.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.2.1
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.3.2
	gopkg.in/resty.v1 v1.12.0
	helm.sh/helm/v3 v3.0.2
	k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894

	k8s.io/klog v1.0.0
	k8s.io/kubernetes v1.14.8
	sigs.k8s.io/controller-runtime v0.2.2
)

replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	// Kubernetes 1.14.8
	k8s.io/kubernetes => k8s.io/kubernetes v1.14.8
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.2.2
)
