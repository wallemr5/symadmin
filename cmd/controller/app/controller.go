package app

import (
	"github.com/spf13/cobra"
	"k8s.io/klog"
)

func NewControllerCmd(cli *DksCli) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "controller",
		Aliases: []string{"ctl"},
		Short:   "Manage controller Component",
		Run: func(cmd *cobra.Command, args []string) {
			klog.Info("enter controller run")
		},
	}

	return cmd
}
