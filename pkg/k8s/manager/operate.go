package manager

import (
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BaseCluster ...
type BaseCluster interface {
	GetPod(opts *client.ListOptions, clusters ...string) (*corev1.Pod, error)
	GetPods(opts *client.ListOptions, clusters ...string) ([]*corev1.Pod, error)
	GetDeployment(opts *client.ListOptions, clusters ...string) ([]*appv1.Deployment, error)
	GetService(opts *client.ListOptions, clusters ...string) ([]*corev1.Service, error)
	GetEndpoint(opts *client.ListOptions, clusters ...string) ([]*corev1.Endpoints, error)
	GetEvent(opts *client.ListOptions, clusters ...string) ([]*corev1.Event, error)
	RestartPods(opts *client.ListOptions, clusters ...string) error
	DeletePod(opts *client.ListOptions, clusters ...string) ([]*corev1.Pod, error)
}

// CustomeCluster ...
type CustomeCluster interface {
	BaseCluster
	GetHelmRelease(cluster, appName, group, releaseName, zone string) (*rls.ListReleasesResponse, error)
}
