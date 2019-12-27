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
	controller "gitlab.dmall.com/arch/sym-admin/pkg/controllers"
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"

	"k8s.io/klog"
	// ctrl "sigs.k8s.io/controller-runtime"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"gitlab.dmall.com/arch/sym-admin/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	logger = logf.KBLog.WithName("controller")
)

func NewControllerCmd(cli *DksCli) *cobra.Command {
	opt := manager.DefaultManagerOption()
	cmd := &cobra.Command{
		Use:     "controller",
		Aliases: []string{"ctl"},
		Short:   "Manage controller Component",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := cli.GetK8sConfig()
			if err != nil {
				klog.Fatalf("unable to get kubeconfig err: %v", err)
			}

			rp := time.Second * 120
			mgr, err := ctrlmanager.New(cfg, ctrlmanager.Options{
				Scheme:             k8sclient.GetScheme(),
				MetricsBindAddress: "0",
				LeaderElection:     opt.EnableLeaderElection,
				// Port:               9443,
				SyncPeriod: &rp,
			})
			if err != nil {
				klog.Fatalf("unable to new manager err: %v", err)
			}

			stopCh := signals.SetupSignalHandler()
			dksMgr, err := manager.NewDksManager(cli.GetKubeInterfaceOrDie(), opt, logger, "controller")
			if err != nil {
				klog.Fatalf("unable to NewDksManager err: %v", err)
			}

			// add http server Runnable
			mgr.Add(dksMgr.Router)

			if dksMgr.K8sMgr != nil {
				// add k8s cluster manager Runnable
				mgr.Add(dksMgr.K8sMgr)
			}
			// Setup all Controllers
			klog.Info("Setting up controller")
			if err := controller.AddToManager(mgr, dksMgr); err != nil {
				klog.Fatalf("unable to register controllers to the manager err: %v", err)
			}

			logger.WithValues("SyncPeriod", rp)
			klog.Info("starting manager")
			if err := mgr.Start(stopCh); err != nil {
				klog.Fatalf("problem start running manager err: %v", err)
			}
		},
	}

	cmd.PersistentFlags().IntVar(&opt.GoroutineThreshold, "goroutine-threshold", opt.GoroutineThreshold, "the max Goroutine Threshold")
	cmd.PersistentFlags().StringVar(&opt.HttpAddr, "http-addr", opt.HttpAddr, "HttpAddr for some info")
	cmd.PersistentFlags().BoolVar(&opt.EnableLeaderElection, "enable-leader", opt.EnableLeaderElection,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	cmd.PersistentFlags().BoolVar(&opt.GinLogEnabled, "enable-ginlog", opt.GinLogEnabled, "Enabled will open gin run log.")
	cmd.PersistentFlags().BoolVar(&opt.PprofEnabled, "enable-pprof", opt.PprofEnabled, "Enabled will open endpoint for go pprof.")
	cmd.PersistentFlags().BoolVar(&opt.MasterEnabled, "enable-master", opt.MasterEnabled, "Enable master controller")
	cmd.PersistentFlags().BoolVar(&opt.WorkerEnabled, "enable-worker", opt.WorkerEnabled, "Enable worker controller")
	return cmd
}
