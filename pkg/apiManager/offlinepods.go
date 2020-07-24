package apiManager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DEFAULT_NAMESPACE string = "sym-admin"
)

func (m *APIManager) GetAllOfflineApp(c *gin.Context) {
	lb := labels.Set{
		"controllerOwner": "offlinePod",
	}

	listOptions := &client.ListOptions{Namespace: DEFAULT_NAMESPACE, LabelSelector: lb.AsSelector()}

	offlineApp := make([]string, 0, 10)
	ctx := context.Background()

	cmlist := &corev1.ConfigMapList{}
	client := m.K8sMgr.MasterClient.GetClient()
	err := client.List(ctx, cmlist, listOptions)

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
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}

	raw, ok := cm.Data["offlineList"]
	if !ok {
		klog.Info("no applist ")
		return
	}
	fmt.Println(raw)
	jerr := json.Unmarshal([]byte(raw), &apps)
	if jerr != nil {
		klog.Errorf("failed to Unmarshal err: %v", jerr)
		AbortHTTPError(c, http.StatusNotFound, "", jerr)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
		"resultMap": gin.H{
			"info": apps,
		},
	})

}
