package v2

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apimanager/model"
)

// GetClusters returns all cluster's status.
func (m *Manager) GetClusters(c *gin.Context) {
	clusterName := c.Param("clusterCode")
	clusters := m.ClustersMgr.GetAll(clusterName)

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
