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
)

var (
	apps []*model.OfflinePod
)

const (
	DEFAULT_NAMESPACE string = "sym-admin"
)

// /api/appname/:appname/offlinepodlist
func (m *APIManager) GetOfflinePods(c *gin.Context) {
	cmname := c.Param("appname")
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
	c.IndentedJSON(http.StatusOK, apps)
}
