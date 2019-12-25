package manager

import (
	"time"

	"github.com/go-logr/logr"

	"gitlab.dmall.com/arch/sym-admin/pkg/healthcheck"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/router"
)

type ManagerOption struct {
	Threadiness        int
	GoroutineThreshold int
	ResyncPeriod       time.Duration
	Features           []string

	// use expose /metrics, /read, /live, /pprof.
	HttpAddr             string
	EnableLeaderElection bool
	GinLogEnabled        bool
	PprofEnabled         bool
	MasterEnabled        bool
	WorkerEnabled        bool
}

type DksManager struct {
	Opt          *ManagerOption
	K8sMgr       *k8smanager.Manager
	Router       *router.Router
	HealthHander healthcheck.Handler
}

func DefaultManagerOption() *ManagerOption {
	return &ManagerOption{
		HttpAddr:             ":8080",
		GoroutineThreshold:   1000,
		EnableLeaderElection: false,
		GinLogEnabled:        true,
		PprofEnabled:         true,
		MasterEnabled:        false,
		WorkerEnabled:        false,
	}
}

func NewDksManager(opt *ManagerOption, logger logr.Logger, componentName string) (*DksManager, error) {
	routerOptions := &router.RouterOptions{
		GinLogEnabled:    opt.GinLogEnabled,
		MetricsEnabled:   true,
		PprofEnabled:     opt.PprofEnabled,
		Addr:             opt.HttpAddr,
		MetricsPath:      "metrics",
		MetricsSubsystem: componentName,
	}

	healthHander := healthcheck.NewHealthHandler()
	healthHander.AddLivenessCheck("goroutine_threshold",
		healthcheck.GoroutineCountCheck(opt.GoroutineThreshold))

	rt := router.NewRouter(routerOptions)
	rt.AddRoutes("index", router.DefaultRoutes())
	rt.AddRoutes("health", healthHander.Routes())
	// rt.AddRoutes("cluster", mgr.Routes())

	return &DksManager{
		Opt:          opt,
		Router:       rt,
		HealthHander: healthHander,
	}, nil
}

func NewHttpsRouter() *router.Router {
	routerOptions := &router.RouterOptions{
		GinLogEnabled:  true,
		MetricsEnabled: false,
		PprofEnabled:   false,
		Addr:           "0.0.0.0:8443",
		CertFilePath:   "./config/certs/" + "server.pem",
		KeyFilePath:    "./config/certs/" + "server.key",
	}

	rt := router.NewRouter(routerOptions)
	rt.AddRoutes("rt", router.DefaultRoutes())
	return rt
}
