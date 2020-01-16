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

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/healthcheck"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/router"
	"k8s.io/klog"
)

type ManagerOption struct {
	Threadiness        int
	GoroutineThreshold int
	ResyncPeriod       time.Duration
	Features           []string
	Repos              map[string]string

	// use expose /metrics, /read, /live, /pprof.
	HTTPAddr                string
	LeaderElectionNamespace string
	EnableLeaderElection    bool
	GinLogEnabled           bool
	PprofEnabled            bool
	MasterEnabled           bool
	WorkerEnabled           bool
	Debug                   bool
}

type DksManager struct {
	Opt *ManagerOption

	Router       *router.Router
	HealthHander healthcheck.Handler
	K8sMgr       *k8smanager.ClusterManager
}

func DefaultManagerOption() *ManagerOption {
	return &ManagerOption{
		HTTPAddr:                ":8080",
		Threadiness:             1,
		GoroutineThreshold:      1000,
		ResyncPeriod:            30 * time.Minute,
		EnableLeaderElection:    true,
		LeaderElectionNamespace: "sym-admin",
		GinLogEnabled:           true,
		PprofEnabled:            true,
		MasterEnabled:           false,
		WorkerEnabled:           false,
		Repos: map[string]string{
			"dmall": "http://chartmuseum.dmall.com",
		},
	}
}

func NewDksManager(cli k8smanager.MasterClient, opt *ManagerOption, componentName string) (*DksManager, error) {
	routerOptions := &router.Options{
		GinLogEnabled:    opt.GinLogEnabled,
		MetricsEnabled:   true,
		PprofEnabled:     opt.PprofEnabled,
		Addr:             opt.HTTPAddr,
		MetricsPath:      "metrics",
		MetricsSubsystem: componentName,
	}

	healthHander := healthcheck.GetHealthHandler()
	healthHander.AddLivenessCheck("goroutine_threshold",
		healthcheck.GoroutineCountCheck(opt.GoroutineThreshold))

	rt := router.NewRouter(routerOptions)
	rt.AddRoutes("index", rt.DefaultRoutes())
	rt.AddRoutes("health", healthHander.Routes())
	// rt.AddRoutes("cluster", mgr.Routes())

	dksMgr := &DksManager{
		Opt:          opt,
		Router:       rt,
		HealthHander: healthHander,
	}
	if opt.MasterEnabled {
		klog.Info("start init multi cluster manager ... ")
		kMgr, err := k8smanager.NewManager(cli, k8smanager.DefaultClusterManagerOption(false))
		if err != nil {
			klog.Fatalf("unable to new k8s manager err: %v", err)
		}

		dksMgr.K8sMgr = kMgr
		dksMgr.K8sMgr.AddPreInit(func() {
			klog.Infof("preInit manager cluster informer ... ")
			for _, c := range dksMgr.K8sMgr.GetAll() {
				advDeployInformer, _ := c.Cache.GetInformer(&workloadv1beta1.AdvDeployment{})
				dksMgr.HealthHander.AddReadinessCheck(fmt.Sprintf("%s_%s", c.Name, "advDeploy_cache_sync"), func() error {
					if advDeployInformer.HasSynced() {
						return nil
					}
					return fmt.Errorf("cluster:%s AdvDeployment cache not sync", c.Name)
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
