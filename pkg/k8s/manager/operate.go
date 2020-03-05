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
	GetPods(opts *client.ListOptions, clusters ...string) (*corev1.PodList, error)
	GetDeployment(opts *client.ListOptions, clusters ...string) (*appv1.DeploymentList, error)
	GetService(opts *client.ListOptions, clusters ...string) (*corev1.ServiceList, error)
	GetEndpoint(opts *client.ListOptions, clusters ...string) (*corev1.EndpointsList, error)
	GetEvent(opts *client.ListOptions, clusters ...string) (*corev1.EventList, error)
	RestartPods(opts *client.ListOptions, clusters ...string) error
	DeletePod(opts *client.ListOptions, clusters ...string) (*corev1.PodList, error)
}

// CustomeCluster ...
type CustomeCluster interface {
	BaseCluster
	GetHelmRelease(cluster, appName, group, releaseName, zone string) ([]*rls.ListReleasesResponse, error)
}
