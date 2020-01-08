package manager

import (
	"sort"
	"sync"
	"time"

	"fmt"

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
	clusters      []*Cluster
	mu            *sync.RWMutex
	MasterKubecli kubernetes.Interface
	Opt           *ClusterManagerOption
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
		clusters:      make([]*Cluster, 0, 4),
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

func (m *ClusterManager) GetAll(name ...string) []*Cluster {
	m.mu.RLock()
	defer m.mu.RUnlock()

	isAll := true
	var ObserveName string
	if len(name) > 0 {
		if name[0] != "all" {
			ObserveName = name[0]
			isAll = false
		}
	}

	list := make([]*Cluster, 0, 4)
	for _, c := range m.clusters {
		if c.Status == ClusterOffline {
			continue
		}

		if isAll {
			list = append(list, c)
		} else {
			if ObserveName != "" && ObserveName == c.Name {
				list = append(list, c)
				break
			}
		}
	}

	return list
}

func (m *ClusterManager) Add(cluster *Cluster) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clusters = append(m.clusters, cluster)
	sort.Slice(m.clusters, func(i int, j int) bool {
		return m.clusters[i].Name > m.clusters[j].Name
	})

	return nil
}

func (m *ClusterManager) Delete(cluster *Cluster) error {
	if cluster == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	newClusters := make([]*Cluster, 0, 4)
	for _, c := range m.clusters {
		if cluster.Name == c.Name {
			continue
		}

		newClusters = append(newClusters, c)
	}

	m.clusters = newClusters
	return nil
}

func (m *ClusterManager) Get(name string) (*Cluster, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if name == "" || name == "all" {
		return nil, fmt.Errorf("single query not support: %s ", name)
	}

	var findCluster *Cluster
	for _, c := range m.clusters {
		if name == c.Name {
			findCluster = c
			break
		}
	}
	if findCluster == nil {
		return nil, fmt.Errorf("cluster: %s not found", name)
	}

	if findCluster.Status == ClusterOffline {
		return nil, fmt.Errorf("cluster: %s found, but offline", name)
	}

	return findCluster, nil
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
