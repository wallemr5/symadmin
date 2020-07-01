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
	"time"
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
			"success":   false,
			"message":   "can not find offlineApp",
			"resultMap": nil,
		})
		return
	}
	for _, cm := range cmlist.Items {
		offlineApp = append(offlineApp, cm.Name)
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
		"resultMap": gin.H{
			"offlineApps": offlineApp,
		},
	})
}

// /api/namespace/:namespace/appname/:appname/offlinepodlist
func (m *APIManager) GetOfflinePods(c *gin.Context) {
	namespace := c.Param("namespace")
	cmname := c.Param("appname")
	apps := []*model.OfflinePod{}
	client := m.K8sMgr.MasterClient.GetClient()
	ctx := context.Background()
	cm := &corev1.ConfigMap{}

	err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: cmname}, cm)

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
		return
	}

	for i, v := range apps {
		ts := v.OfflineTime.Format("2006-01-02 15:04:05")
		parse, errP := time.Parse("2006-01-02 15:04:05", ts)
		if errP == nil {
			apps[i].OfflineTime = parse
		}
	}

	//c.IndentedJSON(http.StatusOK, apps)
	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
		"resultMap": gin.H{
			"info": apps,
		},
	})

}
