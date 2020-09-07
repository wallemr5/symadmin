/*
Copyright 2020 The dks authors.

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
	"github.com/spf13/cobra"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers"
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"

	"k8s.io/klog"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	logger = logf.Log.WithName("controller")
)

func NewControllerCmd(dksCli *DksCli) *cobra.Command {
	opt := manager.DefaultManagerOption()
	cmd := &cobra.Command{
		Use:     "controller",
		Aliases: []string{"ctl"},
		Short:   "Manage controller Component",
		Run: func(cmd *cobra.Command, args []string) {
			PrintFlags(cmd.Flags())

			cfg, err := dksCli.GetK8sConfig()
			if err != nil {
				klog.Fatalf("unable to get kubeconfig, err: %+v", err)
			}

			// Create a manager for managing all the controllers
			ctrlMgr, err := ctrlmanager.New(cfg, ctrlmanager.Options{
				Scheme:                  k8sclient.GetScheme(),
				MetricsBindAddress:      "0",
				HealthProbeBindAddress:  "0",
				LeaderElection:          opt.EnableLeaderElection,
				LeaderElectionNamespace: opt.LeaderElectionNamespace,
				LeaderElectionID:        "sym-controller-election-helper",
				// Port:               9443,
				SyncPeriod: &opt.ResyncPeriod,
			})
			if err != nil {
				klog.Fatalf("unable to create a controllers manager, err: %v", err)
			}

			masterCli := k8smanager.MasterClient{
				KubeCli: dksCli.GetKubeInterfaceOrDie(), // get the kubernetes client
				Manager: ctrlMgr,
			}

			dksMgr, err := manager.NewDksManager(masterCli, opt, "controller")
			if err != nil {
				klog.Fatalf("unable to create a dks manager, err: %v", err)
			}

			components := &utils.Components{}

			// add http server Runnable to components
			components.Add(dksMgr.Router)

			if dksMgr.ClustersMgr != nil {
				// add k8s cluster manager Runnable to components
				components.Add(dksMgr.ClustersMgr)
			}

			// Setup all Controllers
			klog.Info("Setting up controller")
			if err := controllers.AddToManager(ctrlMgr, dksMgr); err != nil {
				klog.Fatalf("unable to register controllers to the controller manager, err: %v", err)
			}

			stopCh := signals.SetupSignalHandler()
			klog.Infof("start custom components")
			go components.Start(stopCh)

			logger.Info("zap debug", "ResyncPeriod", opt.ResyncPeriod)
			klog.Info("starting manager")
			if err := ctrlMgr.Start(stopCh); err != nil {
				klog.Fatalf("problem start running manager err: %v", err)
			}
		},
	}

	cmd.PersistentFlags().IntVar(&opt.GoroutineThreshold, "goroutine-threshold", opt.GoroutineThreshold, "the max Goroutine Threshold")
	cmd.PersistentFlags().IntVar(&opt.Threadiness, "threadiness", opt.Threadiness, "the max Goroutine for controller")
	cmd.PersistentFlags().DurationVar(&opt.ResyncPeriod, "resync-period", opt.ResyncPeriod, "the max resync period to informer")
	cmd.PersistentFlags().StringVar(&opt.HTTPAddr, "http-addr", opt.HTTPAddr, "HTTPAddr for some info")
	cmd.PersistentFlags().BoolVar(&opt.EnableLeaderElection, "enable-leader", opt.EnableLeaderElection,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	cmd.PersistentFlags().StringVar(&opt.LeaderElectionNamespace, "leader-namespaces", opt.LeaderElectionNamespace,
		"the LeaderElectionNamespace is only one active controller manager.")
	cmd.PersistentFlags().StringVar(&opt.DmallChartRepo, "charts-repo", opt.DmallChartRepo, "the dmall charts repo URL")
	cmd.PersistentFlags().BoolVar(&opt.GinLogEnabled, "enable-ginlog", opt.GinLogEnabled, "Enabled will open gin run log.")
	cmd.PersistentFlags().BoolVar(&opt.PprofEnabled, "enable-pprof", opt.PprofEnabled, "Enabled will open endpoint for go pprof.")
	cmd.PersistentFlags().BoolVar(&opt.MasterEnabled, "enable-master", opt.MasterEnabled, "Enable master controller")
	cmd.PersistentFlags().BoolVar(&opt.WorkerEnabled, "enable-worker", opt.WorkerEnabled, "Enable worker controller")
	cmd.PersistentFlags().BoolVar(&opt.ClusterEnabled, "enable-cluster", opt.ClusterEnabled, "Enable cluster controller")
	cmd.PersistentFlags().BoolVar(&opt.OfflinePodEnabled, "enable-offlinepod", opt.OfflinePodEnabled, "Enable offline pod controller")
	cmd.PersistentFlags().BoolVar(&opt.EventEnabled, "enable-event", opt.EventEnabled, "Enable event exporter controller")
	cmd.PersistentFlags().StringVar(&opt.AlertEndpoint, "alert-endpoint", opt.AlertEndpoint, "the alertmanager endpoint URL")
	cmd.PersistentFlags().BoolVar(&opt.Recover, "recover", opt.Recover, "Enable recover function")
	cmd.PersistentFlags().BoolVar(&opt.Debug, "debug", opt.Debug, "Debug mode")
	return cmd
}
