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
	"flag"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func AddFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
}

// PrintFlags logs the flags in the flagset
func PrintFlags(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		klog.Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}

func runHelp(cmd *cobra.Command, args []string) {
	cmd.Help()
}

// GetRootCmd returns the root of the cobra command-tree.
func GetRootCmd(args []string) *cobra.Command {
	opt := DefaultRootOption()
	rootCmd := &cobra.Command{
		Use:               "sym",
		Short:             "Request a new project",
		SilenceUsage:      true,
		DisableAutoGenTag: true,
		Run:               runHelp,
	}

	rootCmd.SetArgs(args)
	rootCmd.PersistentFlags().StringVarP(&opt.Kubeconfig, "kubeconfig", "c", "", "Kubernetes configuration file")
	rootCmd.PersistentFlags().StringVar(&opt.ConfigContext, "context", "", "The name of the kubeconfig context to use")
	rootCmd.PersistentFlags().StringVarP(&opt.Namespace, "namespace", "n", opt.Namespace, "Config namespace")
	rootCmd.PersistentFlags().BoolVarP(&opt.DevelopmentMode, "devel-mode", "d", opt.DevelopmentMode, "Set development mode (mainly for logging)")

	// Make sure that klog logging variables are initialized so that we can
	// update them from this file.
	klog.InitFlags(nil)

	// opts := zap.Options{}
	// opts.BindFlags(flag.CommandLine)
	// log := zap.New(zap.UseFlagOptions(&opts))
	// logf.SetLogger(log)
	logf.SetLogger(zap.New(zap.UseDevMode(opt.DevelopmentMode)))

	// Make sure klog (used by the client-go dependency) logs to stderr, as it
	// will try to log to directories that may not exist in the cilium-operator
	// container (/tmp) and cause the cilium-operator to exit.
	flag.Set("logtostderr", "true")

	AddFlags(rootCmd)
	cli := NewDksCli(opt)

	rootCmd.AddCommand(NewControllerCmd(cli))
	// rootCmd.AddCommand(NewOperatorCmd(cli))
	rootCmd.AddCommand(NewCmdVersion(cli))
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
