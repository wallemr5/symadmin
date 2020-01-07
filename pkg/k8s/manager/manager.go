package manager

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

type ClusterManagerOption struct {
	Namespace     string
	LabelSelector kblabels.Set
}

type ClusterManager struct {
	clusters      map[string]*Cluster
	mu            *sync.RWMutex
	MasterKubecli kubernetes.Interface
	Opt           *ClusterManagerOption
	sclusters     []*Cluster
}

func DefaultClusterManagerOption() *ClusterManagerOption {
	return &ClusterManagerOption{
		Namespace: "default",
		LabelSelector: kblabels.Set{
			"ClusterOwer": "sym-admin",
		},
	}
}

func NewManager(kubecli kubernetes.Interface, log logr.Logger, opt *ClusterManagerOption) (*ClusterManager, error) {
	lsel := opt.LabelSelector.AsSelector()
	configMaps, err := kubecli.CoreV1().ConfigMaps(opt.Namespace).List(metav1.ListOptions{LabelSelector: lsel.String()})
	if err != nil {
		klog.Errorf("unable to get cluster configmap err: %v", err)
	}

	klog.Infof("find %d cluster form namespace: %s ls: %v ", len(configMaps.Items), opt.Namespace, opt.LabelSelector)
	mgr := &ClusterManager{
		MasterKubecli: kubecli,
		clusters:      make(map[string]*Cluster),
		mu:            &sync.RWMutex{},
		Opt:           opt,
	}

	for i := range configMaps.Items {
		cm := &configMaps.Items[i]
		for k, v := range cm.Data {
			if k != "kubeconfig.yaml" {
				klog.Infof("data key:%s", k)
				break
			}

			cluster, err := NewCluster(cm.Name, []byte(v), log)
			if err != nil {
				klog.Errorf("new cluster err: %v", err)
				break
			}

			klog.Infof("add cluster name: %s ", cm.Name)
			mgr.Add(cluster)
		}
	}

	return mgr, nil
}

func (m *ClusterManager) GetAll() map[string]*Cluster {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.clusters
}

func (m *ClusterManager) GetAllSort() []*Cluster {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.sclusters
}

func (m *ClusterManager) Add(cluster *Cluster) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clusters[cluster.GetName()] = cluster

	m.sclusters = append(m.sclusters, cluster)
	sort.Slice(m.sclusters, func(i int, j int) bool {
		return m.sclusters[i].Name > m.sclusters[j].Name
	})

	return nil
}

func (m *ClusterManager) Delete(cluster *Cluster) error {
	if cluster == nil {
		return nil
	}

	if m.clusters[cluster.GetName()] == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.clusters, cluster.GetName())

	return nil
}

func (m *ClusterManager) Get(name string) (*Cluster, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cluster := m.clusters[name]
	if cluster == nil {
		return nil, errors.New("cluster not found:" + name)
	}

	return cluster, nil
}

// InitStart init start Informers
func (m *ClusterManager) InitStart(stopCh <-chan struct{}) error {
	klog.Info("initStart cluster manager ... ")
	for _, c := range m.clusters {
		if !c.healthCheck() {
			break
		}

		c.StartCache(stopCh)
	}
	return nil
}

func (m *ClusterManager) cluterCheck() {
	klog.V(4).Infof("new time: %v", time.Now())
}

// Start timer check cluster health
func (m *ClusterManager) Start(stopCh <-chan struct{}) error {
	klog.Info("start cluster manager check loop ... ")
	wait.Until(m.cluterCheck, time.Minute, stopCh)
	return nil
}
