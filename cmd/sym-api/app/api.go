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
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"

	"k8s.io/klog"
	// ctrl "sigs.k8s.io/controller-runtime"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	logger = logf.KBLog.WithName("controller")
)

func NewApiCmd(cli *DksCli) *cobra.Command {
	opt := apiManager.DefaultApiManagerOption()
	cmd := &cobra.Command{
		Use:     "api",
		Aliases: []string{"api"},
		Short:   "Manage sym api server",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := cli.GetK8sConfig()
			if err != nil {
				klog.Fatalf("unable to get kubeconfig err: %v", err)
			}

			rp := time.Second * 120
			mgr, err := ctrlmanager.New(cfg, ctrlmanager.Options{
				Scheme:             k8sclient.GetScheme(),
				MetricsBindAddress: "0",
				LeaderElection:     false,
				// Port:               9443,
				SyncPeriod: &rp,
			})
			if err != nil {
				klog.Fatalf("unable to new manager err: %v", err)
			}

			stopCh := signals.SetupSignalHandler()
			apiMgr, err := apiManager.NewApiManager(cli.GetKubeInterfaceOrDie(), opt, logger, "controller")
			if err != nil {
				klog.Fatalf("unable to NewDksManager err: %v", err)
			}

			// add http server Runnable
			mgr.Add(apiMgr.Router)

			// add k8s cluster manager Runnable
			mgr.Add(apiMgr.K8sMgr)

			logger.Info("zap debug", "SyncPeriod", rp)
			klog.Info("starting manager")
			if err := mgr.Start(stopCh); err != nil {
				klog.Fatalf("problem start running manager err: %v", err)
			}
		},
	}

	cmd.PersistentFlags().IntVar(&opt.GoroutineThreshold, "goroutine-threshold", opt.GoroutineThreshold, "the max Goroutine Threshold")
	cmd.PersistentFlags().StringVar(&opt.HttpAddr, "http-addr", opt.HttpAddr, "HttpAddr for some info")
	cmd.PersistentFlags().BoolVar(&opt.GinLogEnabled, "enable-ginlog", opt.GinLogEnabled, "Enabled will open gin run log.")
	cmd.PersistentFlags().BoolVar(&opt.PprofEnabled, "enable-pprof", opt.PprofEnabled, "Enabled will open endpoint for go pprof.")
	return cmd
}