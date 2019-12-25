package app

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

type RootOption struct {
	Kubeconfig       string
	ConfigContext    string
	Namespace        string
	DefaultNamespace string
}

func DefaultRootOption() *RootOption {
	return &RootOption{
		Namespace: corev1.NamespaceAll,
	}
}

type DksCli struct {
	RootCmd *cobra.Command
	Opt     *RootOption
}

func NewDksCli(opt *RootOption) *DksCli {
	return &DksCli{
		Opt: opt,
	}
}

func (c *DksCli) GetK8sConfig() (*rest.Config, error) {
	config, err := k8sclient.GetConfigWithContext(c.Opt.Kubeconfig, c.Opt.ConfigContext)
	if err != nil {
		return nil, errors.Wrap(err, "could not get k8s config")
	}

	return config, nil
}

func (c *DksCli) GetKubeInterface() (kubernetes.Interface, error) {
	cfg, err := c.GetK8sConfig()
	if err != nil {
		return nil, errors.Wrap(err, "could not get k8s config")
	}

	kubeCli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("failed to get kubernetes Clientset: %v", err)
	}

	return kubeCli, nil
}
