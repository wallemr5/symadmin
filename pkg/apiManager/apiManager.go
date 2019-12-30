package apiManager

import (
	"context"
	"time"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	"gitlab.dmall.com/arch/sym-admin/pkg/healthcheck"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/router"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ApiManagerOption struct {
	Threadiness        int
	GoroutineThreshold int
	ResyncPeriod       time.Duration
	Features           []string

	// use expose /metrics, /read, /live, /pprof, /api.
	HttpAddr      string
	GinLogEnabled bool
	PprofEnabled  bool
}

type ApiManager struct {
	Opt *ApiManagerOption

	Router       *router.Router
	HealthHander healthcheck.Handler
	K8sMgr       *k8smanager.ClusterManager
}

func DefaultApiManagerOption() *ApiManagerOption {
	return &ApiManagerOption{
		HttpAddr:           ":8080",
		GoroutineThreshold: 1000,
		GinLogEnabled:      true,
		PprofEnabled:       true,
	}
}

func NewApiManager(kubecli kubernetes.Interface, opt *ApiManagerOption, logger logr.Logger, componentName string) (*ApiManager, error) {
	healthHander := healthcheck.NewHealthHandler()
	healthHander.AddLivenessCheck("goroutine_threshold",
		healthcheck.GoroutineCountCheck(opt.GoroutineThreshold))

	apiMgr := &ApiManager{
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
		Addr:             opt.HttpAddr,
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

// Routes
func (m *ApiManager) Routes() []*router.Route {
	var routes []*router.Route

	apiRoutes := []*router.Route{
		{"GET", "/api/cluster/:name", m.GetClusters, ""},
		{"GET", "/api/cluster/:name/appPod/:appName", m.GetPod, ""},
	}

	routes = append(routes, apiRoutes...)
	return routes
}

// GetClusters
func (m *ApiManager) GetClusters(c *gin.Context) {
	clusters := m.K8sMgr.GetAll()

	stas := make([]*model.ClusterStatus, 0, 4)
	for _, c := range clusters {
		stas = append(stas, &model.ClusterStatus{
			Name:   c.Name,
			Status: string(c.Status),
		})
	}

	c.JSON(http.StatusOK, stas)
}

// GetClusters
func (m *ApiManager) GetPod(c *gin.Context) {
	// clusterNmae := c.Param("name")

	appNmae := c.Param("appName")

	clusters := m.K8sMgr.GetAll()

	ctx := context.Background()
	pods := make([]*model.Pod, 0, 4)

	listOptions := &client.ListOptions{}
	listOptions.MatchingLabels(map[string]string{
		"app": appNmae,
	})
	for _, cluster := range clusters {
		if cluster.Status == k8smanager.ClusterOffline {
			continue
		}

		podList := &corev1.PodList{}
		err := cluster.Client.List(ctx, listOptions, podList)
		if err != nil {

			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Error(err, "failed to get pods")
			break
		}

		for i := range podList.Items {
			pod := &podList.Items[i]
			pods = append(pods, &model.Pod{
				Name:   pod.Name,
				NodeIp: pod.Status.HostIP,
				PodIp:  pod.Status.PodIP,
			})
		}

	}

	c.JSON(http.StatusOK, pods)
}
