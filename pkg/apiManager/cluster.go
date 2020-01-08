package apiManager

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
)

// GetClusters ...
func (m *APIManager) GetClusters(c *gin.Context) {
	clusters := m.K8sMgr.GetAll()

	status := make([]*model.ClusterStatus, 0, 4)
	for _, c := range clusters {
		status = append(status, &model.ClusterStatus{
			Name:   c.Name,
			Status: string(c.Status),
		})
	}

	c.JSON(http.StatusOK, status)
}
