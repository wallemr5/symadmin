package apiManager

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetWarningEvents is an interface to help Cancan get out of trouble:)
func (m *APIManager) GetWarningEvents(c *gin.Context) {
	clusterName := c.Param("name")
	namespace := c.Param("namespace")
	appName := c.DefaultQuery("appName", "all")
	group := c.DefaultQuery("group", "")
	clusters := m.K8sMgr.GetAll(clusterName)

	options := make(map[string]string)
	if group != "" {
		options["sym-group"] = group
	}
	if appName != "all" {
		options["app"] = appName
	}

	podOptions := &client.ListOptions{Namespace: namespace}
	podOptions.MatchingLabels(options)
	podList, err := m.Cluster.GetPods(podOptions, clusterName)
	if err != nil {
		klog.Error(err, "failed to get pod list")
		AbortHTTPError(c, GetPodError, "", err)
		return
	}

	result := []*model.Event{}
	for _, cluster := range clusters {
		ctx := context.Background()
		listOptions := &client.ListOptions{Namespace: namespace}
		listOptions.MatchingField("type", corev1.EventTypeWarning)
		events := &corev1.EventList{}

		err := cluster.Client.List(ctx, listOptions, events)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Error(err, "failed to get events")
			AbortHTTPError(c, GetPodEventError, "", err)
			return
		}

	EventLoop:
		for i := range events.Items {
			event := &events.Items[i]
			if event.InvolvedObject.Kind == "AdvDeployment" &&
				event.InvolvedObject.Name == appName {
				for _, e := range result {
					if e.ObjectKind == "AdvDeployment" {
						continue EventLoop
					}
				}
				item := &model.Event{
					ClusterName: cluster.GetName(),
					Namespace:   event.GetNamespace(),
					ObjectName:  event.InvolvedObject.Name,
					ObjectKind:  event.InvolvedObject.Kind,
					Type:        event.Type,
					Count:       event.Count,
					FirstTime:   formatTime(event.FirstTimestamp.String()),
					LastTime:    formatTime(event.LastTimestamp.String()),
					Message:     event.Message,
					Reason:      event.Reason,
				}
				result = append(result, item)
				continue
			}
			if event.InvolvedObject.Kind == "Pod" &&
				isEventBelongPod(podList, event.InvolvedObject.Name) {
				for _, e := range result {
					if e.ObjectKind == "Pod" {
						continue EventLoop
					}
				}
				item := &model.Event{
					ClusterName: cluster.GetName(),
					Namespace:   event.GetNamespace(),
					ObjectName:  event.InvolvedObject.Name,
					ObjectKind:  event.InvolvedObject.Kind,
					Type:        event.Type,
					Count:       event.Count,
					FirstTime:   formatTime(event.FirstTimestamp.String()),
					LastTime:    formatTime(event.LastTimestamp.String()),
					Message:     event.Message,
					Reason:      event.Reason,
				}
				result = append(result, item)
			}
		}
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   nil,
		"resultMap": gin.H{"events": result},
	})
}

func isEventBelongPod(podList []*corev1.Pod, podName string) bool {
	for _, pod := range podList {
		if pod.GetName() == podName {
			return true
		}
	}
	return false
}
