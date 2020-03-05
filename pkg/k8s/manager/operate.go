package manager

import (
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	rlsv2 "k8s.io/helm/pkg/proto/hapi/release"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BaseCluster ...
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
	RestartPods(opts *client.ListOptions, clusters ...string) error
	DeletePod(opts *types.NamespacedName, clusters ...string) error
}

// CustomeCluster ...
type CustomeCluster interface {
	BaseCluster
	GetHelmRelease(opts map[string]string, clusters ...string) ([]*rlsv2.Release, error)
}
