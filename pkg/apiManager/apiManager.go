package apiManager

import (
	"context"
	"fmt"
	"time"

	apiv1 "gitlab.dmall.com/arch/sym-admin/pkg/apiManager/v1"
	apiv2 "gitlab.dmall.com/arch/sym-admin/pkg/apiManager/v2"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/healthcheck"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"gitlab.dmall.com/arch/sym-admin/pkg/router"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

// APIManager ...
type APIManager struct {
	Opt           *Option
	Cluster       k8smanager.CustomeCluster
	Router        *router.Router
	HealthHandler healthcheck.Handler
	K8sMgr        *k8smanager.ClusterManager
}

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

	v1 := apiv1.Manager{}
	v2 := apiv2.Manager{}

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
	rt.AddRoutes("cluster", v1.Routes())
	rt.AddRoutes("cluster", v2.Routes())
	apiMgr.Router = rt
	apiMgr.K8sMgr = k8sMgr
	v1.K8sMgr = k8sMgr
	v2.K8sMgr = k8sMgr
	apiMgr.K8sMgr.AddPreInit(func() {
		klog.Infof("preInit manager cluster informer ... ")
		for _, c := range apiMgr.K8sMgr.GetAll() {
			apiMgr.registryResource(c)
		}
	})

	if opt.IsMeta {
		apiMgr.Cluster = k8sMgr
		v1.Cluster = k8sMgr
		v2.Cluster = k8sMgr
	} else {
		// apiMgr.Cluster = cli
	}

	go apiMgr.ClusterChange()

	return apiMgr, nil
}

func (m *APIManager) registryResource(cluster *k8smanager.Cluster) error {
	healthHandler := healthcheck.GetHealthHandler()
	clusterName := cluster.Name
	advDeployInformer, _ := cluster.Cache.GetInformer(context.TODO(), &workloadv1beta1.AdvDeployment{})
	healthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "advDeploy_cache_sync"), func() error {
		if advDeployInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s AdvDeployment cache not sync", clusterName)
	})

	podInformer, _ := cluster.Cache.GetInformer(context.TODO(), &corev1.Pod{})
	healthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "pod_cache_sync"), func() error {
		if podInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s pod cache not sync", clusterName)
	})

	deploymentInformer, _ := cluster.Cache.GetInformer(context.TODO(), &appsv1.Deployment{})
	healthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "deployment_cache_sync"), func() error {
		if deploymentInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s deployment cache not sync", clusterName)
	})

	statefulSetInformer, _ := cluster.Cache.GetInformer(context.TODO(), &appsv1.StatefulSet{})
	healthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "statefulset_cache_sync"), func() error {
		if statefulSetInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s statefulset cache not sync", clusterName)
	})

	nodeInformer, _ := cluster.Cache.GetInformer(context.TODO(), &corev1.Node{})
	healthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "node_cache_sync"), func() error {
		if nodeInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s node cache not sync", clusterName)
	})

	serviceInformer, _ := cluster.Cache.GetInformer(context.TODO(), &corev1.Service{})
	healthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", clusterName, "service_cache_sync"), func() error {
		if serviceInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s service cache not sync", clusterName)
	})

	cluster.Cache.GetInformer(context.TODO(), &corev1.Event{})
	cluster.Cache.GetInformer(context.TODO(), &corev1.Endpoints{})
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
