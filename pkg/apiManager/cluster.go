package apiManager

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	"k8s.io/klog"
)

// GetClusters returns all cluster's status.
func (m *APIManager) GetClusters(c *gin.Context) {
	clusterName := c.Param("name")
	clusters := m.K8sMgr.GetAll(clusterName)

	status := make([]*model.ClusterStatus, 0, 4)
	for _, c := range clusters {
		status = append(status, &model.ClusterStatus{
			Name:   c.Name,
			Status: string(c.Status),
		})
	}
	sort.Slice(status, func(i, j int) bool {
		return status[i].Name < status[j].Name
	})

	c.IndentedJSON(http.StatusOK, status)
}

// GetClusterResource ...
func (m *APIManager) GetClusterResource(c *gin.Context) {
	clusterName := c.Param("name")
	appName := c.Param("appName")
	clusters := m.K8sMgr.GetAll(clusterName)
	namespace := c.Param("namespace")
	ldcLabel := c.DefaultQuery("ldcLabel", "")
	group := c.DefaultQuery("group", "")

	services, err := getService(clusters, appName, group)
	if err != nil {
		klog.Error(err, "failed to get service")
		AbortHTTPError(c, GetServiceError, "", err)
		return
	}

	pods, err := getPodListByAppName(clusters, appName, group, ldcLabel, namespace)
	if err != nil {
		klog.Error(err, "failed to get pods")
		AbortHTTPError(c, GetPodError, "", err)
		return
	}

	deployments, err := getDeployments(clusters, namespace, appName, group, "")
	if err != nil {
		klog.Errorf("failed to get deployments: %v", err)
		AbortHTTPError(c, GetDeploymentError, "", err)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"services": services,
		"pods":     pods,
		"deploys":  deployments,
	})
}
