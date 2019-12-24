package app

import (
	"github.com/spf13/cobra"
	controller "gitlab.dmall.com/arch/sym-admin/pkg/controllers"
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

type OperatorOptions struct {
	SyncPeriod time.Duration
}

func DefaultOperatorOptions() *OperatorOptions {
	return &OperatorOptions{}
}

func NewOperatorCmd(cli *DksCli) *cobra.Command {
	opt := DefaultOperatorOptions()

	cmd := &cobra.Command{
		Use:     "operator",
		Aliases: []string{"op"},
		Short:   "Manage operator Component",

		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := cli.GetK8sConfig()
			if err != nil {
				klog.Fatalf("unable to get kubeconfig err: %v", err)
			}

			mgr, err := ctrl.NewManager(cfg, ctrl.Options{
				Scheme:             k8sclient.GetScheme(),
				MetricsBindAddress: "0",
				LeaderElection:     cli.Opt.EnableLeaderElection,
				Port:               9443,
			})
			if err != nil {
				klog.Fatalf("unable to new manager err: %v", err)
			}

			// Setup all Controllers
			klog.Info("Setting up controller")
			if err := controller.AddToManager(mgr); err != nil {
				klog.Fatalf("unable to register controllers to the manager err: %v", err)
			}

			// +kubebuilder:scaffold:builder
			klog.Info("starting manager")
			if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
				klog.Fatalf("problem start running manager err: %v", err)
			}
		},
	}

	cmd.PersistentFlags().DurationVarP(&opt.SyncPeriod, "sync-period", "n", opt.SyncPeriod, " sync time")
	return cmd
}
