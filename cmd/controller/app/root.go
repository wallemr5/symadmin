package app

import (
	"flag"
	"github.com/spf13/cobra"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
}

// GetRootCmd returns the root of the cobra command-tree.
func GetRootCmd(args []string) *cobra.Command {
	opt := DefaultRootOption()
	rootCmd := &cobra.Command{
		Use:               "sym",
		Short:             "Request a new project",
		SilenceUsage:      true,
		DisableAutoGenTag: true,
	}

	rootCmd.SetArgs(args)
	rootCmd.PersistentFlags().StringVarP(&opt.Kubeconfig, "kubeconfig", "c", "", "Kubernetes configuration file")
	rootCmd.PersistentFlags().StringVar(&opt.ConfigContext, "context", "", "The name of the kubeconfig context to use")
	rootCmd.PersistentFlags().StringVarP(&opt.Namespace, "namespace", "n", opt.Namespace, "Config namespace")
	rootCmd.PersistentFlags().StringVarP(&opt.HttpAddr, "addr", "", opt.HttpAddr, "HttpAddr for some info")
	rootCmd.PersistentFlags().BoolVarP(&opt.EnableLeaderElection, "enableLeader", "l", opt.EnableLeaderElection,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	// Make sure that klog logging variables are initialized so that we can
	// update them from this file.
	klog.InitFlags(nil)
	ctrl.SetLogger(zap.Logger(true))

	// Make sure klog (used by the client-go dependency) logs to stderr, as it
	// will try to log to directories that may not exist in the cilium-operator
	// container (/tmp) and cause the cilium-operator to exit.
	flag.Set("logtostderr", "true")

	AddFlags(rootCmd)
	cli := NewDksCli(opt)

	rootCmd.AddCommand(NewControllerCmd(cli))
	rootCmd.AddCommand(NewOperatorCmd(cli))
	return rootCmd
}

func hideInheritedFlags(orig *cobra.Command, hidden ...string) {
	orig.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		for _, hidden := range hidden {
			_ = cmd.Flags().MarkHidden(hidden) // nolint: errcheck
		}

		orig.SetHelpFunc(nil)
		orig.HelpFunc()(cmd, args)
	})
}
