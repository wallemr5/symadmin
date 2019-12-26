module gitlab.dmall.com/arch/sym-admin

go 1.13

require (
	github.com/DeanThompson/ginpprof v0.0.0-20190408063150-3be636683586
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible
	github.com/chai2010/gettext-go v0.0.0-20191225085308-6b9f4b1008e1 // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/gin-gonic/gin v1.5.0
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible
	github.com/goph/emperror v0.17.2
	github.com/huandu/xstrings v1.2.1 // indirect
	github.com/jmoiron/sqlx v1.2.0 // indirect
	github.com/lestrrat-go/backoff v1.0.0
	github.com/lib/pq v1.3.0 // indirect
	github.com/microcosm-cc/bluemonday v1.0.2
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/openkruise/kruise v0.3.0
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.2.1
	github.com/rubenv/sql-migrate v0.0.0-20191213152630-06338513c237 // indirect
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.3.2
	gopkg.in/resty.v1 v1.12.0
	k8s.io/api v0.17.0
	k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/code-generator v0.17.0
	k8s.io/helm v2.16.1+incompatible

	k8s.io/klog v1.0.0
	k8s.io/kubectl v0.17.0 // indirect
	k8s.io/kubernetes v1.14.8 // indirect
	sigs.k8s.io/controller-runtime v0.2.2
	vbom.ml/util v0.0.0-20180919145318-efcd4e0f9787 // indirect
)

replace (
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	// Kubernetes 1.14.8
	k8s.io/kubernetes => k8s.io/kubernetes v1.14.8
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.2.2
)
