package apiManager

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	"gitlab.dmall.com/arch/sym-admin/pkg/helm/object"
	"k8s.io/klog"
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

// LintHelmTemplate ...
func (m *APIManager) LintHelmTemplate(c *gin.Context) {
	rlsName := c.PostForm("rlsName")
	ns := c.PostForm("namespace")
	overrideValue := c.PostForm("overrideValue")
	chartPkg, header, err := c.Request.FormFile("chart")
	if err != nil {
		klog.Errorf("upload chart file error: %+v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	klog.Infof("get upload chart file: %s", header.Filename)

	chartByte, err := ioutil.ReadAll(chartPkg)
	if err != nil {
		klog.Errorf("read chart file error: +v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
	}

	_, err = object.RenderTemplate(chartByte, rlsName, ns, overrideValue)
	if err != nil {
		klog.Errorf("lint helm template error: %+v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
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
