package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetWarningEvents is an interface to help Cancan get out of trouble:)
func (m *Manager) GetWarningEvents(c *gin.Context) {
	clusterName := c.Param("name")
	namespace := c.Param("namespace")
	appName, ok := c.GetQuery("appName")
	if !ok {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("can not get appName"))
		return
	}
	group, ok := c.GetQuery("group")
	if !ok {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("can not get group"))
		return
	}

	clusters := m.K8sMgr.GetAll(clusterName)
	advList := []*workloadv1beta1.AdvDeployment{}
	for _, cluster := range clusters {
		adv := &workloadv1beta1.AdvDeployment{}
		err := cluster.Client.Get(context.TODO(), types.NamespacedName{
			Namespace: namespace,
			Name:      appName,
		}, adv)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Errorf("get advdeployment error: %v", err)
		}
		advList = append(advList, adv)
	}

	runningCount := 0
	for _, adv := range advList {
		if adv.Status.AggrStatus.Status == workloadv1beta1.AppStatusRuning {
			runningCount++
		}
	}

	if runningCount == len(advList) {
		c.IndentedJSON(http.StatusOK, gin.H{
			"success":   true,
			"message":   nil,
			"resultMap": gin.H{"events": []*model.Event{}},
		})
		return
	}

	lb := labels.Set{
		"app":       appName,
		"sym-group": group,
	}
	podOptions := &client.ListOptions{Namespace: namespace, LabelSelector: lb.AsSelector()}

	podList, err := m.Cluster.GetPods(podOptions, clusterName)
	if err != nil {
		klog.Error(err, "failed to get pod list")
		AbortHTTPError(c, GetPodError, "", err)
		return
	}
	podMap := make(map[string]*corev1.Pod)
	for _, s := range podList {
		podMap[s.Name] = s
	}

	result := []*model.Event{}
	for _, cluster := range clusters {
		ctx := context.Background()

		fd := fields.Set{
			"type": corev1.EventTypeWarning,
		}
		listOptions := &client.ListOptions{Namespace: namespace, FieldSelector: fd.AsSelector()}

		events := &corev1.EventList{}
		err := cluster.Client.List(ctx, events, listOptions)
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
					FirstTime:   utils.FormatTime(event.FirstTimestamp.String()),
					LastTime:    utils.FormatTime(event.LastTimestamp.String()),
					Message:     event.Message,
					Reason:      event.Reason,
				}
				result = append(result, item)
				continue
			}

			_, ok := podMap[event.InvolvedObject.Name]
			if event.InvolvedObject.Kind == "Pod" && ok {
				for i, e := range result {
					if e.ObjectKind == "Pod" && e.Count >= event.Count {
						continue EventLoop
					} else if e.ObjectKind == "Pod" && e.Count < event.Count {
						result = append(result[:i], result[i+1:]...)
					}
				}
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
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   nil,
		"resultMap": gin.H{"events": result},
	})
}
