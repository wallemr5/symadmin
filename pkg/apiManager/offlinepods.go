package apiManager

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DEFAULT_NAMESPACE string = "sym-admin"
)

func (m *APIManager) GetAllOfflineApp(c *gin.Context) {
	options := make(map[string]string)
	options["controllerOwner"] = "offlinePod"
	listOptions := &client.ListOptions{Namespace: DEFAULT_NAMESPACE}
	listOptions.MatchingLabels(options)
	offlineApp := make([]string, 0, 10)
	ctx := context.Background()

	cmlist := &corev1.ConfigMapList{}
	client := m.K8sMgr.MasterClient.GetClient()
	err := client.List(ctx, listOptions, cmlist)

	if err != nil {
		c.IndentedJSON(GetConfigMapError, gin.H{
			"success":     "false",
			"message":     "can not find offlineApp",
			"offlineApps": nil,
		})
		return
	}
	for _, cm := range cmlist.Items {
		offlineApp = append(offlineApp, cm.Name)
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"success":     "success",
		"message":     nil,
		"offlineApps": offlineApp,
	})
}

// /api/appname/:appname/offlinepodlist
func (m *APIManager) GetOfflinePods(c *gin.Context) {
	cmname := c.Param("appname")
	apps := []*model.OfflinePod{}
	client := m.K8sMgr.MasterClient.GetClient()
	ctx := context.Background()
	cm := &corev1.ConfigMap{}

	err := client.Get(ctx, types.NamespacedName{Namespace: DEFAULT_NAMESPACE, Name: cmname}, cm)

	if err != nil {
		klog.Errorf("get app error %v: ", err)
		AbortHTTPError(c, GetConfigMapError, "", err)
		return
	}

	raw, ok := cm.Data["offlineList"]
	if !ok {
		klog.Info("no applist ")
		return
	}

	jerr := json.Unmarshal([]byte(raw), &apps)
	if jerr != nil {
		klog.Errorf("failed to Unmarshal err: %v", jerr)
		AbortHTTPError(c, http.StatusNotFound, "", jerr)
	}
	//c.IndentedJSON(http.StatusOK, apps)
	c.IndentedJSON(http.StatusBadRequest, gin.H{
		"success": false,
		"message": err.Error(),
		"resultMap": gin.H{
			"path":   apps,
			"applog": "",
		},
	})

}
