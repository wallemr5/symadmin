package v2

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/portforwarder"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/klog"
)

// Client encapsulates a Helm Client and a Tunnel for that client to interact with the Tiller pod
type Client struct {
	Name string
	*kube.Tunnel
	*helm.Client
}

// NewClient
func NewClient(kubeConfig []byte) (*Client, error) {
	config, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create kubernetes client config for helm client")
	}

	client, err := k8sclient.NewClientFromConfig(config)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to create kubernetes client for helm client")
	}

	return NewClientFromConfig(config, client, "")
}

// NewClientFromConfig
func NewClientFromConfig(config *rest.Config, client kubernetes.Interface, name string) (*Client, error) {
	klog.V(4).Infof("create kubernetes tunnel name: %s", name)
	tillerTunnel, err := portforwarder.New("kube-system", client, config)
	if err != nil {
		return nil, errors.Wrapf(err, "cluster:%s failed to create kubernetes tunnel", name)
	}

	tillerTunnelAddress := fmt.Sprintf("localhost:%d", tillerTunnel.Local)
	klog.V(4).Infof("created kubernetes tunnel on address:%s", tillerTunnelAddress)

	hClient := helm.NewClient(helm.Host(tillerTunnelAddress))

	return &Client{Tunnel: tillerTunnel, Client: hClient, Name: name}, nil
}

// SaveChartByte save a struct chart to []byte
func SaveChartByte(c *chart.Chart) ([]byte, error) {
	filename, err := chartutil.Save(c, "/tmp")
	if err != nil {
		klog.Infof("err:%#v", err)
		return nil, errors.Wrap(err, "save tmp chart fail")
	}
	defer os.Remove(filename)

	chartByte, err := ioutil.ReadFile(filename)
	if err != nil {
		klog.Infof("err:%#v", err)
		return nil, errors.Wrap(err, "read tmp chart to byte fail")
	}

	klog.V(4).Infof("name:%s, filename:%s", c.GetMetadata().Name, filename)
	return chartByte, nil
}
