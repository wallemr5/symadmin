package apiManager

import (
	"fmt"
	"time"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/healthcheck"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"gitlab.dmall.com/arch/sym-admin/pkg/router"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// Option ...
type Option struct {
	Threadiness        int
	GoroutineThreshold int
	IsMeta             bool
	ResyncPeriod       time.Duration
	Features           []string

	// use expose /metrics, /read, /live, /pprof, /api.
	HTTPAddr       string
	GinLogEnabled  bool
	GinLogSkipPath []string
	PprofEnabled   bool
}

// APIManager ...
type APIManager struct {
	Opt *Option

	Cluster       k8smanager.CustomeCluster
	Router        *router.Router
	HealthHandler healthcheck.Handler
	K8sMgr        *k8smanager.ClusterManager
}

// DefaultOption ...
func DefaultOption() *Option {
	return &Option{
		HTTPAddr:           ":8080",
		IsMeta:             true,
		GoroutineThreshold: 1000,
		GinLogSkipPath:     []string{"/ready", "/live"},
		GinLogEnabled:      true,
		PprofEnabled:       true,
	}
}

// NewAPIManager ...
func NewAPIManager(cli k8smanager.MasterClient, opt *Option, componentName string) (*APIManager, error) {
	healthHandler := healthcheck.GetHealthHandler()
	healthHandler.AddLivenessCheck("goroutine_threshold",
		healthcheck.GoroutineCountCheck(opt.GoroutineThreshold))

	apiMgr := &APIManager{
		Opt:           opt,
		HealthHandler: healthHandler,
	}

	klog.Info("start init multi cluster manager ... ")
	k8sMgr, err := k8smanager.NewManager(cli, k8smanager.DefaultClusterManagerOption(true, labels.GetClusterLs()))
	if err != nil {
		klog.Fatalf("unable to new k8s manager err: %v", err)
	}

	routerOptions := &router.Options{
		GinLogEnabled:    opt.GinLogEnabled,
		GinLogSkipPath:   opt.GinLogSkipPath,
		MetricsEnabled:   true,
		PprofEnabled:     opt.PprofEnabled,
		Addr:             opt.HTTPAddr,
		MetricsPath:      "metrics",
		MetricsSubsystem: componentName,
	}
	rt := router.NewRouter(routerOptions)
	rt.AddRoutes("index", rt.DefaultRoutes())
	rt.AddRoutes("health", healthHandler.Routes())
	rt.AddRoutes("cluster", apiMgr.Routes())
	apiMgr.Router = rt
	apiMgr.K8sMgr = k8sMgr
	apiMgr.K8sMgr.AddPreInit(func() {
		klog.Infof("preInit manager cluster informer ... ")
		for _, c := range apiMgr.K8sMgr.GetAll() {
			apiMgr.registryResource(c)
		}
	})

	if opt.IsMeta {
		apiMgr.Cluster = k8sMgr
	} else {
		// apiMgr.Cluster = cli
	}

	go apiMgr.ClusterChange()

	return apiMgr, nil
}

func (m *APIManager) registryResource(cluster *k8smanager.Cluster) error {
	healthHandler := healthcheck.GetHealthHandler()
	clusterName := cluster.Name
	advDeployInformer, _ := cluster.Cache.GetInformer(&workloadv1beta1.AdvDeployment{})
	healthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "advDeploy_cache_sync"), func() error {
		if advDeployInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s AdvDeployment cache not sync", clusterName)
	})

	podInformer, _ := cluster.Cache.GetInformer(&corev1.Pod{})
	healthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "pod_cache_sync"), func() error {
		if podInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s pod cache not sync", clusterName)
	})

	deploymentInformer, _ := cluster.Cache.GetInformer(&appsv1.Deployment{})
	healthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "deployment_cache_sync"), func() error {
		if deploymentInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s deployment cache not sync", clusterName)
	})

	statefulSetInformer, _ := cluster.Cache.GetInformer(&appsv1.StatefulSet{})
	healthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "statefulset_cache_sync"), func() error {
		if statefulSetInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s statefulset cache not sync", clusterName)
	})

	nodeInformer, _ := cluster.Cache.GetInformer(&corev1.Node{})
	healthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "node_cache_sync"), func() error {
		if nodeInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s node cache not sync", clusterName)
	})

	serviceInformer, _ := cluster.Cache.GetInformer(&corev1.Service{})
	healthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "service_cache_sync"), func() error {
		if serviceInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s service cache not sync", clusterName)
	})

	cluster.Cache.GetInformer(&corev1.Event{})
	cluster.Cache.GetInformer(&corev1.Endpoints{})
	return nil
}

// ClusterChange ...
func (m *APIManager) ClusterChange() {
	for list := range m.K8sMgr.ClusterAddInfo {
		for name := range list {
			cluster, err := m.K8sMgr.Get(name)
			if err != nil {
				klog.Errorf("get cluster[%s] faile: %+v", cluster.Name, err)
				break
			}
			m.registryResource(cluster)
		}
	}
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
			Path:    "/api/cluster/:name/namespace/:namespace/app/:appName/resource",
			Handler: m.GetClusterResource,
			Desc:    GetClusterResourceDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/appPods/labels",
			Handler: m.GetPodByLabels,
			Desc:    GetPodByLabelsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/appPod/:appName/helm",
			Handler: m.GetHelmReleases,
			Desc:    GetHelmReleasesDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/helm/:releaseName",
			Handler: m.GetHelmReleaseInfo,
			Desc:    GetHelmReleaseInfoDesc,
		},
		{
			Method:  "POST",
			Path:    "/api/cluster/:name/namespace/:namespace/app/:appName/restart",
			Handler: m.DeletePodByGroup,
			Desc:    DeletePodByGroupDesc,
		},
		{
			Method:  "DELETE",
			Path:    "/api/cluster/:name/namespace/:namespace/pod/:podName",
			Handler: m.DeletePodByName,
			Desc:    DeletePodByNameDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/pod/:podName",
			Handler: m.GetPodByName,
			Desc:    GetPodByNameDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/endpoints/:appName",
			Handler: m.GetEndpoints,
			Desc:    GetEndpointsDesc,
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
			Path:    "/api/cluster/:name/namespace/:namespace/services/:appName",
			Handler: m.GetServices,
			Desc:    GetServicesDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/service/:svcName",
			Handler: m.GetServiceInfo,
			Desc:    GetServiceInfoDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/deployments/:appName",
			Handler: m.GetDeployments,
			Desc:    GetDeploymentsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/deployment/:deployName",
			Handler: m.GetDeploymentInfo,
			Desc:    GetDeploymentInfoDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/deployments/stat",
			Handler: m.GetDeploymentsStat,
			Desc:    GetDeploymentsStatDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/pods/:podName/event",
			Handler: m.GetPodEvent,
			Desc:    GetPodEventDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/events/warning",
			Handler: m.GetWarningEvents,
			Desc:    GetWarningEventsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/pod/logfiles",
			Handler: m.GetFiles,
			Desc:    GetFilesDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/pods/:podName/logs",
			Handler: m.HandleLogs,
			Desc:    HandleLogsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/pods/:podName/logs/file",
			Handler: m.HandleFileLogs,
			Desc:    HandleFileLogsDesc,
		},
	}

	routes = append(routes, apiRoutes...)
	return routes
}
