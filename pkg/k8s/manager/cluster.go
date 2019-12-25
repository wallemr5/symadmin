package manager

import (
	"github.com/go-logr/logr"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Cluster struct {
	Name              string
	RawKubeconfig     []byte
	Meta              map[string]string
	RestConfig        *rest.Config
	ctrlRuntimeClient client.Client
	dynamicClient     dynamic.Interface

	log logr.Logger
}

func NewCluster(name string, kubeconfig []byte, log logr.Logger) (*Cluster, error) {
	cluster := &Cluster{
		Name:          name,
		RawKubeconfig: kubeconfig,
		log:           log.WithValues("cluster", name),
	}

	err := cluster.initK8SClients()
	if err != nil {
		return nil, emperror.Wrap(err, "could not re-init k8s clients")
	}

	return cluster, nil
}

func (c *Cluster) GetName() string {
	return c.Name
}

func (c *Cluster) getRestConfig(kubeconfig []byte) (*rest.Config, error) {
	clusterConfig, err := clientcmd.Load(kubeconfig)
	if err != nil {
		return nil, errors.Wrapf(err, "could not load kubeconfig name: %s", c.Name)
	}

	cfg, err := clientcmd.NewDefaultClientConfig(*clusterConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "could not create k8s rest config")
	}

	return cfg, nil
}

func (c *Cluster) initK8SClients() error {
	restConfig, err := c.getRestConfig(c.RawKubeconfig)
	if err != nil {
		return emperror.Wrap(err, "could not get k8s rest config")
	}
	c.RestConfig = restConfig

	ctrlRuntimeClient, err := c.getCtrlRuntimeClient(restConfig)
	if err != nil {
		return errors.Wrap(err, "could not get control-runtime client")
	}
	c.ctrlRuntimeClient = ctrlRuntimeClient

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return errors.Wrap(err, "could not get dynamic client")
	}
	c.dynamicClient = dynamicClient

	return nil
}

func (c *Cluster) getCtrlRuntimeClient(config *rest.Config) (client.Client, error) {
	writeObj, err := client.New(config, client.Options{})
	if err != nil {
		return nil, emperror.Wrap(err, "could not create control-runtime client")
	}

	return client.DelegatingClient{
		Reader: &client.DelegatingReader{
			CacheReader:  writeObj,
			ClientReader: writeObj,
		},
		Writer:       writeObj,
		StatusClient: writeObj,
	}, nil
}
