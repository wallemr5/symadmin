package manager

import (
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
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
	Client        client.Client
	Kubecli       kubernetes.Interface
	DynamicClient dynamic.Interface

	log             logr.Logger
	Mgr             manager.Manager
	Cache           cache.Cache
	internalStopper chan struct{}

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
		return nil, errors.Wrapf(err, "could not re-init k8s clients name:%s", name)
	}

	return cluster, nil
}

func (c *Cluster) GetName() string {
	return c.Name
}

func (c *Cluster) initK8SClients() error {
	cfg, err := k8sclient.NewClientConfig(c.RawKubeconfig)
	if err != nil {
		return errors.Wrapf(err, "could not get rest config name:%s", c.Name)
	}
	c.RestConfig = cfg

	dynamicClient, err := dynamic.NewForConfig(c.RestConfig)
	if err != nil {
		return errors.Wrapf(err, "could not new dynamiccli name:%s", c.Name)
	}
	c.DynamicClient = dynamicClient

	kubecli, err := kubernetes.NewForConfig(c.RestConfig)
	if err != nil {
		return errors.Wrapf(err, "could not new kubecli name:%s", c.Name)
	}

	c.Kubecli = kubecli
	rp := time.Minute * 5
	o := manager.Options{
		Scheme:     k8sclient.GetScheme(),
		SyncPeriod: &rp,
	}

	mgr, err := manager.New(c.RestConfig, o)
	if err != nil {
		return errors.Wrapf(err, "could not new manager name:%s", c.Name)
	}

	c.Mgr = mgr
	c.Client = mgr.GetClient()
	c.Cache = mgr.GetCache()
	return nil
}

func (c *Cluster) healthCheck() bool {
	body, err := c.Kubecli.Discovery().RESTClient().Get().AbsPath("/healthz").Do().Raw()
	if err != nil {
		runtime.HandleError(errors.Wrapf(err, "Failed to do cluster health check for cluster %q", c.Name))
		c.Status = ClusterOffline
		return false
	}

	if !strings.EqualFold(string(body), "ok") {
		c.Status = ClusterOffline
		return false
	}
	c.Status = ClusterReady
	return true
}

func (c *Cluster) StartCache(stopCh <-chan struct{}) {
	if c.Started {
		klog.Infof("cluster name: %s cache Informers is already startd", c.Name)
		return
	}

	klog.Infof("cluster name: %s start cache Informers ", c.Name)
	go func() {
		err := c.Cache.Start(c.internalStopper)
		klog.Warningf("cluster name: %s cache Informers quit end err:%v", c.Name, err)
	}()

	c.Cache.WaitForCacheSync(stopCh)
	c.Started = true
}

func (c *Cluster) Stop() {
	close(c.internalStopper)
}
