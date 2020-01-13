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

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
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
	rt.AddRoutes("index", rt.DefaultRoutes())
	rt.AddRoutes("health", healthHander.Routes())
	rt.AddRoutes("cluster", apiMgr.Routes())
	apiMgr.Router = rt
	return apiMgr, nil
}

// Routes ...
func (m *APIManager) Routes() []*router.Route {
	var routes []*router.Route

	apiRoutes := []*router.Route{
		{
			Method:  "GET",
			Path:    "/api/cluster/:name",
			Handler: m.GetClusters,
			Desc:    GetClusterDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/appPod/:appName",
			Handler: m.GetPod,
			Desc:    GetPodDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/nodeIp/:ip/",
			Handler: m.GetNodeProject,
			Desc:    GetNodeProjectDesc,
		},
		{
			Method:  "DELETE",
			Path:    "/api/cluster/:name/appPod/:appName",
			Handler: m.DeletePodByGroup,
			Desc:    DeletePodByGroupDesc,
		},
		{
			Method:  "DELETE",
			Path:    "/api/cluster/:name/appPod/:appName/pods/:podName",
			Handler: m.DeletePodByName,
			Desc:    DeletePodByNameDesc},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/endpointName/:endpointName",
			Handler: m.GetEndpoints,
			Desc:    GetEndpointsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/nodeName/:nodeName",
			Handler: m.GetNodeInfo,
			Desc:    GetNodeInfoDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/terminal",
			Handler: m.GetTerminal,
			Desc:    GetTerminalDesc},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/service/:appName/",
			Handler: m.GetServices,
			Desc:    GetServicesDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/deployments",
			Handler: m.GetDeployments,
			Desc:    GetDeploymentsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/appPod/:appName/event",
			Handler: m.GetPodEvent,
			Desc:    GetPodEventDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/appPod/:appName/logs",
			Handler: m.HandleLogs,
			Desc:    HandleLogsDesc},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/appPod/:appName/logs/file",
			Handler: m.HandleFileLogs,
			Desc:    HandleFileLogsDesc,
		},
	}

	routes = append(routes, apiRoutes...)
	return routes
}
