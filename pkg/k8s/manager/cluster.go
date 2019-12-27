package manager

import (
	"time"

	"strings"

	"github.com/go-logr/logr"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type ClusterStatusType string

// These are valid status of a cluster.
const (
	// ClusterReady means the cluster is ready to accept workloads.
	ClusterReady ClusterStatusType = "Ready"
	// ClusterOffline means the cluster is temporarily down or not reachable
	ClusterOffline ClusterStatusType = "Offline"
)

type Cluster struct {
	Name          string
	AliasName     string
	RawKubeconfig []byte
	Meta          map[string]string
	RestConfig    *rest.Config
	client        client.Client
	Kubecli       kubernetes.Interface
	dynamicClient dynamic.Interface

	log             logr.Logger
	mgr             manager.Manager
	cache           cache.Cache
	internalStopper chan<- struct{}

	Status ClusterStatusType
	// Started is true if the Informers has been Started
	Started bool
}

func NewCluster(name string, kubeconfig []byte, log logr.Logger) (*Cluster, error) {
	cluster := &Cluster{
		Name:            name,
		RawKubeconfig:   kubeconfig,
		log:             log.WithValues("cluster", name),
		internalStopper: make(chan struct{}),
		Started:         false,
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
	cfg, err := c.getRestConfig(c.RawKubeconfig)
	if err != nil {
		return errors.Wrapf(err, "could not get rest config name:%s", c.Name)
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return errors.Wrapf(err, "could not new dynamiccli name:%s", c.Name)
	}
	c.dynamicClient = dynamicClient

	kubecli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrapf(err, "could not new kubecli name:%s", c.Name)
	}

	c.Kubecli = kubecli
	rp := time.Minute * 5
	o := manager.Options{
		Scheme:     k8sclient.GetScheme(),
		SyncPeriod: &rp,
	}

	mgr, err := manager.New(cfg, o)
	if err != nil {
		return errors.Wrapf(err, "could not new manager name:%s", c.Name)
	}

	c.RestConfig = cfg
	c.mgr = mgr
	c.client = mgr.GetClient()
	c.cache = mgr.GetCache()
	return nil
}

func (c *Cluster) health_check() bool {
	body, err := c.Kubecli.Discovery().RESTClient().Get().AbsPath("/healthz").Do().Raw()
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to do cluster health check for cluster %q", c.Name))
		c.Status = ClusterOffline
		return false
	} else {
		if !strings.EqualFold(string(body), "ok") {
			c.Status = ClusterOffline
			return false
		} else {
			c.Status = ClusterReady
		}
	}

	return true
}
