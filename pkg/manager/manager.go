/*
Copyright 2019 The dks authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package manager

import (
	"time"

	"fmt"

	"context"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/healthcheck"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"gitlab.dmall.com/arch/sym-admin/pkg/router"
	"k8s.io/klog"
)

type ManagerOption struct {
	Threadiness        int
	GoroutineThreshold int
	ResyncPeriod       time.Duration
	Features           []string
	Repos              map[string]string
	DmallChartRepo     string
	AlertEndpoint      string

	// use expose /metrics, /read, /live, /pprof.
	HTTPAddr                string
	LeaderElectionNamespace string
	EnableLeaderElection    bool
	GinLogEnabled           bool
	GinLogSkipPath          []string
	PprofEnabled            bool
	MasterEnabled           bool
	WorkerEnabled           bool
	ClusterEnabled          bool
	OfflinePodEnabled       bool
	EventEnabled            bool
	Debug                   bool
	Recover                 bool
}

type DksManager struct {
	Opt *ManagerOption

	Router        *router.Router
	HealthHandler healthcheck.Handler
	ClustersMgr   *k8smanager.ClusterManager
}

func DefaultManagerOption() *ManagerOption {
	return &ManagerOption{
		HTTPAddr:                ":8080",
		Threadiness:             1,
		GoroutineThreshold:      1000,
		ResyncPeriod:            2 * time.Hour,
		EnableLeaderElection:    true,
		LeaderElectionNamespace: "sym-admin",
		GinLogEnabled:           true,
		GinLogSkipPath:          []string{"/ready", "/live"},
		PprofEnabled:            true,
		MasterEnabled:           false,
		WorkerEnabled:           false,
		ClusterEnabled:          false,
		OfflinePodEnabled:       false,
		EventEnabled:            false,
		Debug:                   false,
		Recover:                 false,
		DmallChartRepo:          "",
		Repos: map[string]string{
			"dmall": "http://chartmuseum.dmall.com",
		},
	}
}

// NewDksManager is used for managing a couple of components of dks.
func NewDksManager(masterCli k8smanager.MasterClient, opt *ManagerOption, componentName string) (*DksManager, error) {
	routerOptions := &router.Options{
		GinLogEnabled:    opt.GinLogEnabled,
		MetricsEnabled:   true,
		GinLogSkipPath:   opt.GinLogSkipPath,
		PprofEnabled:     opt.PprofEnabled,
		Addr:             opt.HTTPAddr,
		MetricsPath:      "metrics",
		MetricsSubsystem: componentName,
	}

	healthHandler := healthcheck.GetHealthHandler()
	healthHandler.AddLivenessCheck("goroutine_threshold",
		healthcheck.GoroutineCountCheck(opt.GoroutineThreshold))

	rt := router.NewRouter(routerOptions)
	rt.AddRoutes("index", rt.DefaultRoutes())
	rt.AddRoutes("health", healthHandler.Routes())
	// rt.AddRoutes("cluster", mgr.Routes())
	if opt.DmallChartRepo != "" {
		opt.Repos["dmall"] = opt.DmallChartRepo
	}

	dksMgr := &DksManager{
		Opt:           opt,
		Router:        rt,
		HealthHandler: healthHandler,
	}
	if opt.MasterEnabled || opt.ClusterEnabled || opt.OfflinePodEnabled {
		klog.Info("start to initialize these multi managers of every cluster ... ")
		clustersMgr, err := k8smanager.NewClusterManager(masterCli, k8smanager.DefaultClusterManagerOption(false, labels.GetClusterLs()))
		if err != nil {
			klog.Fatalf("unable to create a new clusters manager, err: %v", err)
		}

		dksMgr.ClustersMgr = clustersMgr
		dksMgr.ClustersMgr.AddPreInit(func() {
			klog.Infof("preInit manager cluster informer ... ")
			for _, c := range dksMgr.ClustersMgr.GetAll() {
				advDeployInformer, _ := c.Cache.GetInformer(context.TODO(), &workloadv1beta1.AdvDeployment{})
				dksMgr.HealthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", c.Name, "advDeploy_cache_sync"), func() error {
					if advDeployInformer.HasSynced() {
						return nil
					}
					return fmt.Errorf("cluster:%s AdvDeployment cache hasn't be synchronized", c.Name)
				})
			}
		})
	}

	return dksMgr, nil
}

func NewHttpsRouter() *router.Router {
	routerOptions := &router.Options{
		GinLogEnabled:  true,
		MetricsEnabled: false,
		PprofEnabled:   false,
		Addr:           "0.0.0.0:8443",
		CertFilePath:   "./config/certs/" + "server.pem",
		KeyFilePath:    "./config/certs/" + "server.key",
	}

	rt := router.NewRouter(routerOptions)
	rt.AddRoutes("rt", rt.DefaultRoutes())
	return rt
}
