package apiManager

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
)

// GetHelmReleases ...
func (m *APIManager) GetHelmReleases(c *gin.Context) {
	// clusterName := c.Param("name")
	// appName := c.Param("appName")
	// group := c.DefaultQuery("group", "")
	// clusters := m.K8sMgr.GetAll(clusterName)
	// zone := c.DefaultQuery("symZone", "")

	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
	})
}

// GetHelmReleaseInfo ...
func (m *APIManager) GetHelmReleaseInfo(c *gin.Context) {
	// clusterName := c.Param("name")
	// releaseName := c.Param("releaseName")
	// cluster, err := m.K8sMgr.Get(clusterName)
	// zone := c.DefaultQuery("symZone", "")
	// if err != nil {
	// 	c.IndentedJSON(http.StatusBadRequest, gin.H{
	// 		"success":   false,
	// 		"message":   err.Error(),
	// 		"resultMap": nil,
	// 	})
	// 	return
	// }

	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
	})
}

func getGroupFromHelmRelease(name string) model.GroupEnum {
	switch {
	case strings.Contains(name, "blue"):
		return model.BlueGroup
	case strings.Contains(name, "green"):
		return model.GreenGroup
	case strings.Contains(name, "canary"):
		return model.CanaryGroup
	case strings.Contains(name, "svc"):
		return model.SvcGroup
	default:
		return model.Unkonwn
	}
}
