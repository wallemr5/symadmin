package manager

import (
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
)

// ClusterInterface ...
type ClusterInterface interface {
	GetPod(clusterName, namespace, podName string) (*model.Pod, error)
	GetPods(clusterName, namespace, appName, group, ldcLabel, zone string) ([]*model.Pod, error)
	GetDeployment(clusterName, namespace, appName, group, ldcLabel, zone string) ([]*model.DeploymentInfo, error)
	GetService(clusterName, namespace, appName, group string) ([]*model.ServiceInfo, error)
	GetEndpoint(clusterName, namespace, appName string) ([]*model.Endpoint, error)
	GetHelm(clusterName, appName, group, zone string) ([]*model.HelmRelease, error)
	GetHelmInfo(clusterName, zone, releaseName string) ([]*model.HelmWholeRelease, error)
	GetTerminal(clusterName, namespace, podName, containerName, cmd, option, ws string) error
	GetEvent(clusterName, namespace, podName, limit string) ([]*model.Event, error)
	RestartPods(clusterName, namespace, appName, group, ldcLabel, zone string) error
	DeletePod(clusterName, namespace, podName string) error
}
