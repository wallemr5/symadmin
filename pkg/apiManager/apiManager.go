package apiManager

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"gitlab.dmall.com/arch/sym-admin/pkg/healthcheck"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/router"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// Option ...
type Option struct {
	Threadiness        int
	GoroutineThreshold int
	ResyncPeriod       time.Duration
	Features           []string

	// use expose /metrics, /read, /live, /pprof, /api.
	HTTPAddr      string
	GinLogEnabled bool
	PprofEnabled  bool
}

// APIManager ...
type APIManager struct {
	Opt *Option

	Router       *router.Router
	HealthHander healthcheck.Handler
	K8sMgr       *k8smanager.ClusterManager
}

// DefaultOption ...
func DefaultOption() *Option {
	return &Option{
		HTTPAddr:           ":8080",
		GoroutineThreshold: 1000,
		GinLogEnabled:      true,
		PprofEnabled:       true,
	}
}

// NewAPIManager ...
func NewAPIManager(kubecli kubernetes.Interface, opt *Option, logger logr.Logger, componentName string) (*APIManager, error) {
	healthHander := healthcheck.NewHealthHandler()
	healthHander.AddLivenessCheck("goroutine_threshold",
		healthcheck.GoroutineCountCheck(opt.GoroutineThreshold))

	apiMgr := &APIManager{
		Opt:          opt,
		HealthHander: healthHander,
	}

	klog.Info("start init multi cluster manager ... ")
	manager, err := k8smanager.NewManager(kubecli, logger, k8smanager.DefaultClusterManagerOption())
	if err != nil {
		klog.Fatalf("unable to new k8s manager err: %v", err)
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	manager.InitStart(ctx.Done())
	apiMgr.K8sMgr = manager

	routerOptions := &router.RouterOptions{
		GinLogEnabled:    opt.GinLogEnabled,
		MetricsEnabled:   true,
		PprofEnabled:     opt.PprofEnabled,
		Addr:             opt.HTTPAddr,
		MetricsPath:      "metrics",
		MetricsSubsystem: componentName,
	}
	rt := router.NewRouter(routerOptions)
	rt.AddRoutes("index", router.DefaultRoutes())
	rt.AddRoutes("health", healthHander.Routes())
	rt.AddRoutes("cluster", apiMgr.Routes())
	apiMgr.Router = rt
	return apiMgr, nil
}

// Routes ...
func (m *APIManager) Routes() []*router.Route {
	var routes []*router.Route

	apiRoutes := []*router.Route{
		{"GET", "/api/cluster/:name", m.GetClusters, ""},
		{"GET", "/api/cluster/:name/appPod/:appName", m.GetPod, ""},
		//{"GET", "/api/cluster/:name/podIp/:ip/", m.GetPodProject, ""},
		{"GET", "/api/cluster/:name/nodeIp/:ip/", m.GetNodeProject, ""},
		{"GET", "/api/cluster/:name/endpointName/:endpointName/", m.GetEndpoints, ""},
		{"GET", "/api/cluster/:name/nodeName/:nodeName/", m.GetNodeInfo, ""},
		{"GET", "/api/cluster/:name/terminal", m.GetTerminal, ""},
		{"GET", "/api/cluster/:name/service/:appName/", m.GetServices, ""},
		{"GET", "/api/cluster/:name/deployments", m.GetDeployments, ""},
		{"GET", "/api/cluster/:name/appPod/:appName/event", m.GetPodEvent, ""},
	}

	routes = append(routes, apiRoutes...)
	return routes
}
