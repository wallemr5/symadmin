package apiManager

import (
	"fmt"
	"time"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/healthcheck"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/router"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
func NewAPIManager(mgr manager.Manager, opt *Option, componentName string) (*APIManager, error) {
	healthHander := healthcheck.NewHealthHandler()
	healthHander.AddLivenessCheck("goroutine_threshold",
		healthcheck.GoroutineCountCheck(opt.GoroutineThreshold))

	apiMgr := &APIManager{
		Opt:          opt,
		HealthHander: healthHander,
	}

	klog.Info("start init multi cluster manager ... ")
	k8sMgr, err := k8smanager.NewManager(mgr, k8smanager.DefaultClusterManagerOption())
	if err != nil {
		klog.Fatalf("unable to new k8s manager err: %v", err)
	}

	routerOptions := &router.Options{
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
	apiMgr.K8sMgr = k8sMgr
	apiMgr.K8sMgr.AddPreInit(func() {
		klog.Infof("preInit manager cluster informer ... ")
		for _, c := range apiMgr.K8sMgr.GetAll() {
			clusterName := c.Name
			advDeployInformer, _ := c.Cache.GetInformer(&workloadv1beta1.AdvDeployment{})
			apiMgr.HealthHander.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "advDeploy_cache_sync"), func() error {
				if advDeployInformer.HasSynced() {
					return nil
				}
				return fmt.Errorf("cluster:%s AdvDeployment cache not sync", clusterName)
			})

			podInformer, _ := c.Cache.GetInformer(&corev1.Pod{})
			apiMgr.HealthHander.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "pod_cache_sync"), func() error {
				if podInformer.HasSynced() {
					return nil
				}
				return fmt.Errorf("cluster:%s pod cache not sync", clusterName)
			})

			deploymentInformer, _ := c.Cache.GetInformer(&appsv1.Deployment{})
			apiMgr.HealthHander.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "deployment_cache_sync"), func() error {
				if deploymentInformer.HasSynced() {
					return nil
				}
				return fmt.Errorf("cluster:%s deployment cache not sync", clusterName)
			})

			statefulSetInformer, _ := c.Cache.GetInformer(&appsv1.StatefulSet{})
			apiMgr.HealthHander.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "statefulset_cache_sync"), func() error {
				if statefulSetInformer.HasSynced() {
					return nil
				}
				return fmt.Errorf("cluster:%s statefulset cache not sync", clusterName)
			})

			nodeInformer, _ := c.Cache.GetInformer(&corev1.Node{})
			apiMgr.HealthHander.AddReadinessCheck(fmt.Sprintf("%s_%s", c.Name, "node_cache_sync"), func() error {
				if nodeInformer.HasSynced() {
					return nil
				}
				return fmt.Errorf("cluster:%s node cache not sync", clusterName)
			})

			serviceInformer, _ := c.Cache.GetInformer(&corev1.Service{})
			apiMgr.HealthHander.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "service_cache_sync"), func() error {
				if serviceInformer.HasSynced() {
					return nil
				}
				return fmt.Errorf("cluster:%s service cache not sync", clusterName)
			})

			c.Cache.GetInformer(&corev1.Event{})
			c.Cache.GetInformer(&corev1.Endpoints{})
		}
	})

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
			Path:    "/api/cluster/:name/nodeIp/:ip",
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
			Desc:    DeletePodByNameDesc,
		},
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
			Desc:    GetTerminalDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/exec",
			Handler: m.ExecOnceWithHTTP,
			Desc:    ExecOnceWithHTTPDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/service/:appName",
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
			Path:    "/api/cluster/:name/appPod/:appName/pods/:podName/event",
			Handler: m.GetPodEvent,
			Desc:    GetPodEventDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/appPod/:appName/pods/:podName/files",
			Handler: m.GetFiles,
			Desc:    GetFilesDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/appPod/:appName/pods/:podName/logs",
			Handler: m.HandleLogs,
			Desc:    HandleLogsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/appPod/:appName/pods/:podName/logs/file",
			Handler: m.HandleFileLogs,
			Desc:    HandleFileLogsDesc,
		},
	}

	routes = append(routes, apiRoutes...)
	return routes
}
