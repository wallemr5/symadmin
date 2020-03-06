package manager

import (
	"errors"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	rlsv2 "k8s.io/helm/pkg/proto/hapi/release"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BaseCluster is used to shield the complexity of the underlying multi-cluster and single cluster.
type BaseCluster interface {
	GetOriginKubeCli(clusters ...string) (kubernetes.Interface, error)
	GetRestConfig(clusters ...string) (*rest.Config, error)
	GetPod(opts *types.NamespacedName, clusters ...string) (*corev1.Pod, error)
	GetPods(opts *client.ListOptions, clusters ...string) ([]*corev1.Pod, error)
	GetNodes(opts *client.ListOptions, clusters ...string) ([]*corev1.Node, error)
	GetDeployment(opts *client.ListOptions, clusters ...string) ([]*appv1.Deployment, error)
	GetService(opts *client.ListOptions, clusters ...string) ([]*corev1.Service, error)
	GetEndpoint(opts *client.ListOptions, clusters ...string) ([]*corev1.Endpoints, error)
	GetEvent(opts *client.ListOptions, clusters ...string) ([]*corev1.Event, error)
	DeletePods(opts *client.ListOptions, clusters ...string) error
	DeletePod(opts *types.NamespacedName, clusters ...string) error
}

// CustomeCluster extend the methods of the basic cluster, including some special business methods.
type CustomeCluster interface {
	BaseCluster
	GetHelmRelease(opts map[string]string, clusters ...string) ([]*rlsv2.Release, error)
}

// GetOriginKubeCli returns the kubecli of the master cluster client if len(clusters) == 0, otherwise
// returns the specific kubecli.
func (m *ClusterManager) GetOriginKubeCli(clusters ...string) (kubernetes.Interface, error) {
	if len(clusters) > 0 {
		if len(clusters) > 1 {
			return nil, errors.New("too many clusters")
		}
		cluster, err := m.Get(clusters[0])
		if err != nil {
			return nil, err
		}
		return cluster.KubeCli, nil
	}
	return m.KubeCli, nil
}

// GetRestConfig returns the rest.Config of the master cluster client if len(clusters) == 0, otherwise
// returns the specific rest.Config.
func (m *ClusterManager) GetRestConfig(clusters ...string) (*rest.Config, error) {
	if len(clusters) > 0 {
		if len(clusters) > 1 {
			return nil, errors.New("too many clusters")
		}
		cluster, err := m.Get(clusters[0])
		if err != nil {
			return nil, err
		}
		return cluster.RestConfig, nil
	}
	return m.Manager.GetConfig(), nil
}
