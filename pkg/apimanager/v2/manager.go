package v2

import (
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
)

// Manager ...
type Manager struct {
	Cluster     k8smanager.CustomizedCluster
	ClustersMgr *k8smanager.ClusterManager
}
