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

package app

import (
	"time"

	"github.com/spf13/cobra"
	"gitlab.dmall.com/arch/sym-admin/pkg/apimanager"
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"
	"k8s.io/klog"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
)

var (
	logger = logf.Log.WithName("controller")
)

// NewAPICmd ...
func NewAPICmd(cli *DksCli) *cobra.Command {
	opt := apimanager.DefaultOption()
	cmd := &cobra.Command{
		Use:     "api",
		Aliases: []string{"api"},
		Short:   "The api server with supporting to interact with multiple clusters",
		Run: func(cmd *cobra.Command, args []string) {
			PrintFlags(cmd.Flags())

			cfg, err := cli.GetK8sConfig()
			if err != nil {
				klog.Fatalf("unable to get kubeconfig, err: %v", err)
			}

			rp := time.Second * 120
			ctrlMgr, err := ctrlmanager.New(cfg, ctrlmanager.Options{
				Scheme:                 k8sclient.GetScheme(),
				MetricsBindAddress:     "0",
				HealthProbeBindAddress: "0",
				LeaderElection:         false,
				// Port:               9443,
				SyncPeriod: &rp,
			})
			if err != nil {
				klog.Fatalf("unable to create a controllers manager, err: %v", err)
			}

			masterCli := k8smanager.MasterClient{
				KubeCli: cli.GetKubeInterfaceOrDie(),
				Manager: ctrlMgr,
			}
			apiMgr, err := apimanager.NewAPIManager(masterCli, opt, "controller")
			if err != nil {
				klog.Fatalf("unable to create the api manager, err: %v", err)
			}

			// add http server Runnable
			ctrlMgr.Add(apiMgr.Router)

			// add k8s cluster manager Runnable
			ctrlMgr.Add(apiMgr.ClustersMgr)

			logger.Info("zap debug", "SyncPeriod", rp)
			klog.Info("starting the controllers manager...")
			stopCh := signals.SetupSignalHandler()
			if err := ctrlMgr.Start(stopCh); err != nil {
				klog.Fatalf("start running controllers manager, err: %v", err)
			}
		},
	}

	cmd.PersistentFlags().IntVar(&opt.GoroutineThreshold, "goroutine-threshold", opt.GoroutineThreshold, "the max Goroutine Threshold")
	cmd.PersistentFlags().StringVar(&opt.HTTPAddr, "http-addr", opt.HTTPAddr, "HttpAddr for some info")
	cmd.PersistentFlags().BoolVar(&opt.IsMeta, "is-meta", opt.IsMeta, "Whether it is a meta cluster")
	cmd.PersistentFlags().BoolVar(&opt.GinLogEnabled, "enable-ginlog", opt.GinLogEnabled, "Enabled will open gin run log.")
	cmd.PersistentFlags().BoolVar(&opt.PprofEnabled, "enable-pprof", opt.PprofEnabled, "Enabled will open endpoint for go pprof.")
	return cmd
}
