package manager

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/healthcheck"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
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

type MasterClient struct {
	KubeCli kubernetes.Interface
	manager.Manager
}

type ClusterManager struct {
	MasterClient
	mu             *sync.RWMutex
	Opt            *ClusterManagerOption
	clusters       []*Cluster
	PreInit        func()
	Started        bool
	clusterAddName chan map[string]string
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
		if status == string(ClusterMaintain) {
			klog.Infof("cluster name: %s status: %s", cm.Name, status)
			return "", false
		}
	}

	if kubeconfig, ok = cm.Data[KeyKubeconfig]; !ok {
		return "", false
	}

	return kubeconfig, true
}

func NewManager(cli MasterClient, opt *ClusterManagerOption, clusterAddName chan map[string]string) (*ClusterManager, error) {
	cMgr := &ClusterManager{
		MasterClient:   cli,
		clusters:       make([]*Cluster, 0, 4),
		mu:             &sync.RWMutex{},
		Opt:            opt,
		clusterAddName: clusterAddName,
	}

	err := cMgr.preStart()
	if err != nil {
		klog.Errorf("preStart cluster err: %v", err)
		return nil, err
	}

	cMgr.Started = true
	return cMgr, nil
}

func (m *ClusterManager) AddPreInit(preInit func()) {
	if m.PreInit != nil {
		klog.Errorf("cluster manager already have preInit func ")
	}

	m.PreInit = preInit
}

// getClusterConfigmap
func (m *ClusterManager) getClusterConfigmap() ([]*corev1.ConfigMap, error) {
	cms := make([]*corev1.ConfigMap, 0, 4)
	if m.Started {
		configmaps := &corev1.ConfigMapList{}
		err := m.Manager.GetClient().List(context.Background(), &client.ListOptions{
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
		for i := range configmaps.Items {
			cms = append(cms, &configmaps.Items[i])
		}

	} else {
		cmList, err := m.KubeCli.CoreV1().ConfigMaps(m.Opt.Namespace).List(metav1.ListOptions{LabelSelector: labels.SelectorFromSet(m.Opt.LabelSelector).String()})
		if err != nil {
			klog.Errorf("unable to get cluster configmap err: %v", err)
		}
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil, err
			}

			klog.Errorf("failed to ConfigMapList ls :%v, err: %v", m.Opt.LabelSelector, err)
			return nil, err
		}

		for i := range cmList.Items {
			cms = append(cms, &cmList.Items[i])
		}
	}

	sort.Slice(cms, func(i, j int) bool {
		return cms[i].Name < cms[j].Name
	})
	return cms, nil
}

// GetAll get all cluster
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

	configmaps, err := m.getClusterConfigmap()
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

	return nil
}

func (m *ClusterManager) cluterCheck() {

	klog.V(4).Info("cluster configmap check.")
	configmaps, err := m.getClusterConfigmap()
	if err != nil {
		klog.Fatalf("unable to get cluster configmap err: %v", err)
	}

	expectList := map[string]string{}
	for _, cm := range configmaps {
		config, _ := convertToKubeconfig(cm)
		expectList[cm.Name] = config
	}

	m.mu.Lock()
	m.mu.Unlock()

	currentList := map[string]*Cluster{}
	for _, c := range m.clusters {
		currentList[c.Name] = c
	}

	newClusters := make([]*Cluster, 0, 4)
	delList := currentList
	addList := map[string]string{}
	for name, conf := range expectList {
		cls, ok := currentList[name]
		if !ok {
			addList[name] = conf
			continue
		}
		delete(delList, name)
		if strings.EqualFold(conf, string(cls.RawKubeconfig)) {
			newClusters = append(newClusters, cls)
			continue
		}
		if conf == "" {
			cls.Status = ClusterMaintain
			newClusters = append(newClusters, cls)
			continue
		}
		delList[name] = cls
		addList[name] = conf
	}

	if len(delList) == 0 && len(addList) == 0 {
		return
	}

	healthHander := healthcheck.GetHealthHandler()
	for _, cls := range delList {
		klog.Infof("delete cluster:%s connect", cls.Name)
		cls.Stop()
		healthHander.RemoveLivenessCheck(fmt.Sprintf("%s_%s", cls.Name, "advDeploy_cache_sync"))
	}

	for name, conf := range addList {
		klog.Infof("create cluster:%s connect", name)
		newcls, err := m.addNewClusters(name, conf)
		if err != nil {
			return
		}
		newClusters = append(newClusters, newcls)
	}

	sort.Slice(newClusters, func(i, j int) bool {
		return newClusters[i].Name > newClusters[j].Name
	})
	m.clusters = newClusters

	m.clusterAddName <- addList
	return
}

func (m *ClusterManager) addNewClusters(name string, kubeconfig string) (*Cluster, error) {
	// config change
	nc, err := NewCluster(name, []byte(kubeconfig), logger)
	if err != nil {
		klog.Errorf("cluster: %s new client err: %v", name, err)
		return nil, err
	}

	klog.V(4).Infof("cluster:%s add AdvDeployment cache", name)
	healthHander := healthcheck.GetHealthHandler()

	advDeployInformer, _ := nc.Cache.GetInformer(&workloadv1beta1.AdvDeployment{})
	healthHander.AddReadinessCheck(fmt.Sprintf("%s_%s", name, "advDeploy_cache_sync"), func() error {
		if advDeployInformer.HasSynced() {
			return nil
		}
		return fmt.Errorf("cluster:%s AdvDeployment cache not sync", name)
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	nc.StartCache(ctx.Done())
	return nc, nil
}

// Start timer check cluster health
func (m *ClusterManager) Start(stopCh <-chan struct{}) error {
	if m.PreInit != nil {
		m.PreInit()
	}
	klog.Info("start cluster manager check loop ... ")
	wait.Until(m.cluterCheck, time.Minute, stopCh)
	return nil
}
