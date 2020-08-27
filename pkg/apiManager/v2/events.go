package v2

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPodEvent return pod event
func (m *Manager) GetPodEvent(c *gin.Context) {
	clusterName := c.Param("clusterCode")
	namespace := c.Param("namespace")
	podName := c.Param("podName")
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	clusters := m.K8sMgr.GetAll(clusterName)

	result := []*model.Event{}
	for _, cluster := range clusters {
		ctx := context.Background()
		listOptions := &client.ListOptions{
			Namespace: namespace,
			Raw:       &metav1.ListOptions{Limit: limit},
		}
		if podName != "all" && podName != "" {
			listOptions.FieldSelector = fields.Set{"podName": podName}.AsSelector()
		}
		events := &corev1.EventList{}

		err := cluster.Client.List(ctx, events, listOptions)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Error(err, "failed to get pod events")
			c.IndentedJSON(http.StatusBadRequest, gin.H{
				"success":   false,
				"message":   err.Error(),
				"resultMap": nil,
			})
			return
		}

		for i := range events.Items {
			event := &events.Items[i]
			item := &model.Event{
				ClusterName: cluster.GetName(),
				Namespace:   event.GetNamespace(),
				ObjectName:  event.InvolvedObject.Name,
				ObjectKind:  event.InvolvedObject.Kind,
				Type:        event.Type,
				Count:       event.Count,
				FirstTime:   utils.FormatTime(event.FirstTimestamp.String()),
				LastTime:    utils.FormatTime(event.LastTimestamp.String()),
				Message:     event.Message,
				Reason:      event.Reason,
			}
			result = append(result, item)
		}
	}
	if len(result) > int(limit) {
		result = result[int64(len(result))-limit:]
	}
	sort.Slice(result, func(i, j int) bool {
		t1, _ := time.Parse("2006-01-02 15:04:05", result[i].LastTime)
		t2, _ := time.Parse("2006-01-02 15:04:05", result[j].LastTime)
		return t1.Before(t2)
	})

	c.IndentedJSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   nil,
		"resultMap": gin.H{"events": result},
	})
}
