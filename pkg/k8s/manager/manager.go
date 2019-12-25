package manager

import (
	"errors"
	"sync"

	"k8s.io/client-go/kubernetes"
)

type Manager struct {
	clusters map[string]*Cluster
	mu       *sync.RWMutex
	kubecli  kubernetes.Interface
}

func NewManager(kubecli kubernetes.Interface) *Manager {
	mgr := &Manager{
		kubecli:  kubecli,
		clusters: make(map[string]*Cluster),
		mu:       &sync.RWMutex{},
	}

	return mgr
}

func (m *Manager) GetAll() map[string]*Cluster {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.clusters
}

func (m *Manager) Add(cluster *Cluster) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clusters[cluster.GetName()] = cluster

	return nil
}

func (m *Manager) Delete(cluster *Cluster) error {
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

func (m *Manager) Get(name string) (*Cluster, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cluster := m.clusters[name]
	if cluster == nil {
		return nil, errors.New("cluster not found")
	}

	return cluster, nil
}

// check cluster headlth
func (c *Manager) Start(stopCh <-chan struct{}) error {
	return nil
}
