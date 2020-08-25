package v1

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	"k8s.io/klog"
)

// GetClusters returns all cluster's status.
func (m *Manager) GetClusters(c *gin.Context) {
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
func (m *Manager) GetClusterResource(c *gin.Context) {
	clusterName := c.Param("name")
	appName := c.Param("appName")
	namespace := c.Param("namespace")
	ldcLabel := c.DefaultQuery("ldcLabel", "")
	group := c.DefaultQuery("group", "")
	zone := c.DefaultQuery("symZone", "")

	var deployments []*model.DeploymentInfo
	deployments, err := m.getDeployments(clusterName, namespace, appName, group, zone, ldcLabel)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}
	if len(deployments) == 0 {
		deployments, err = m.getStatefulset(clusterName, namespace, appName, group, zone, ldcLabel)
		if err != nil {
			c.IndentedJSON(http.StatusBadRequest, gin.H{
				"success":   false,
				"message":   err.Error(),
				"resultMap": nil,
			})
			return
		}
	}

	pods, err := m.getPodListByAppName(clusterName, namespace, appName, group, zone, ldcLabel)
	if err != nil {
		klog.Error(err, "failed to get pods")
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}

	result := gin.H{}
	groups := []string{"blue", "green", "gray"}
	for _, group := range groups {
		var desired, updated, ready, available, unavailable int32
		var deploys []*model.DeploymentInfo
		for _, deployment := range deployments {
			if group == deployment.Group {
				desired += *deployment.DesiredReplicas
				updated += deployment.UpdatedReplicas
				ready += deployment.ReadyReplicas
				available += deployment.AvailableReplicas
				unavailable += deployment.UnavailableReplicas
				deploys = append(deploys, deployment)
			}
		}
		stat := gin.H{
			"desiredReplicas":     desired,
			"updatedReplicas":     updated,
			"readyReplicas":       ready,
			"availableReplicas":   available,
			"unavailableReplicas": unavailable,
			"deploys":             deploys,
		}
		switch group {
		case "blue":
			stat["pods"] = pods["blueGroup"]
			result["blueGroup"] = stat
		case "green":
			stat["pods"] = pods["greenGroup"]
			result["greenGroup"] = stat
		case "gray":
			stat["pods"] = pods["canaryGroup"]
			result["canaryGroup"] = stat
		}
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   nil,
		"resultMap": result,
	})
}
