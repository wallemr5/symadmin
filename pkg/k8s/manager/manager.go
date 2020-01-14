package manager

import (
	"sort"
	"sync"
	"time"

	"fmt"

	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	logger = logf.KBLog.WithName("controller")
)

const (
	KeyKubeconfig = "kubeconfig.yaml"
	KeyStauts     = "status"
)

type ClusterManagerOption struct {
	Namespace     string
	LabelSelector map[string]string
}

type ClusterManager struct {
	clusters  []*Cluster
	mu        *sync.RWMutex
	MasterMgr manager.Manager
	Opt       *ClusterManagerOption
	PreInit   func()
}

func DefaultClusterManagerOption() *ClusterManagerOption {
	return &ClusterManagerOption{
		Namespace: "default",
		LabelSelector: map[string]string{
			"ClusterOwer": "sym-admin",
		},
	}
}

func convertToKubeconfig(cm *corev1.ConfigMap) (string, bool) {
	var kubeconfig string
	var ok bool

	if status, ok := cm.Data[KeyStauts]; ok {
		if status == "maintaining" {
			klog.Infof("cluster name: %s status: %s", cm.Name, status)
			return "", false
		}
	}

	if kubeconfig, ok = cm.Data[KeyKubeconfig]; !ok {
		return "", false
	}

	return kubeconfig, true
}

func NewManager(mgr manager.Manager, opt *ClusterManagerOption) (*ClusterManager, error) {
	cMgr := &ClusterManager{
		MasterMgr: mgr,
		clusters:  make([]*Cluster, 0, 4),
		mu:        &sync.RWMutex{},
		Opt:       opt,
	}

	return cMgr, nil
}

func (m *ClusterManager) AddPreInit(preInit func()) {
	if m.PreInit != nil {
		klog.Errorf("cluster manager already have preInit func ")
	}

	m.PreInit = preInit
}

// getClusterByConfigmap
func (m *ClusterManager) getClusterByConfigmap() ([]*corev1.ConfigMap, error) {
	configmaps := &corev1.ConfigMapList{}
	err := m.MasterMgr.GetClient().List(context.Background(), &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(m.Opt.LabelSelector),
		Namespace:     m.Opt.Namespace,
	}, configmaps)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, err
		}

		klog.Errorf("failed to ConfigMapList ls :%v, err: %v", m.Opt.LabelSelector, err)
		return nil, err
	}

	cms := make([]*corev1.ConfigMap, 0, 4)
	for i := range configmaps.Items {
		cms = append(cms, &configmaps.Items[i])
	}

	sort.Slice(cms, func(i, j int) bool {
		return cms[i].Name < cms[j].Name
	})
	return cms, nil
}

// GetAll
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
	if _, err := m.Get(cluster.Name); err == nil {
		return fmt.Errorf("cluster name: %s is already add to manager", cluster.Name)
	}

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
	m.mu.Lock()
	defer m.mu.Unlock()

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

func (m *ClusterManager) preStart() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	configmaps, err := m.getClusterByConfigmap()
	if err != nil {
		klog.Errorf("unable to get cluster configmap err: %v", err)
		return err
	}

	klog.Infof("find %d cluster form namespace: %s ls: %v ", len(configmaps), m.Opt.Namespace, m.Opt.LabelSelector)
	for _, cm := range configmaps {
		kubeconfig, ok := convertToKubeconfig(cm)
		if !ok {
			klog.Errorf("cluster: %s convertToKubeconfig err: %v", cm.Name, err)
			continue
		}

		c, err := NewCluster(cm.Name, []byte(kubeconfig), logger)
		if err != nil {
			klog.Errorf("cluster: %s new client err: %v", cm.Name, err)
			continue
		}

		if !c.healthCheck() {
			klog.Errorf("cluster: %s check offline", cm.Name)
			continue
		}

		c.StartCache(ctx.Done())
		m.Add(c)
		klog.Infof("add cluster name: %s ", cm.Name)
	}

	if m.PreInit != nil {
		m.PreInit()
	}

	return nil
}

func (m *ClusterManager) cluterCheck() {
	klog.V(4).Infof("new time: %v", time.Now())
}

// Start timer check cluster health
func (m *ClusterManager) Start(stopCh <-chan struct{}) error {
	m.preStart()

	klog.Info("start cluster manager check loop ... ")
	wait.Until(m.cluterCheck, time.Minute, stopCh)
	return nil
}
