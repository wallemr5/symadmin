package v2

import (
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
)

// Manager ...
type Manager struct {
	Cluster k8smanager.CustomeCluster
	K8sMgr  *k8smanager.ClusterManager
}
