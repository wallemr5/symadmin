package apiManager

import (
	"context"
	"net/http"
	"strconv"

	"sort"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetNodeProject ...
func (m *APIManager) GetNodeProject(c *gin.Context) {
	clusterName := c.Param("name")
	nodeName := c.Param("nodeName")

	clusters := m.K8sMgr.GetAll(clusterName)

	nodePro := &model.NodeProjects{
		Projects: make([]*model.Project, 0),
	}

	listOptions := &client.ListOptions{}
	listOptions.MatchingField("spec.nodeName", nodeName)

	ctx := context.Background()
	for _, cluster := range clusters {
		podList := &corev1.PodList{}
		err := cluster.Client.List(ctx, listOptions, podList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Error(err, "failed to get pods")
			AbortHTTPError(c, GetPodError, "", err)
			return
		}

		for i := range podList.Items {
			var ok bool
			var appNmae string
			pod := &podList.Items[i]

			if appNmae, ok = pod.GetLabels()[labels.ObserveMustLabelAppName]; !ok {
				continue
			}

			if _, ok = pod.GetLabels()[labels.ObserveMustLabelGroupName]; !ok {
				continue
			}

			podInfo := &model.Project{
				PodIP: pod.Status.PodIP,
			}

			podInfo.AppName = appNmae
			if domainName, ok := pod.GetLabels()[labels.ObserveMustLabelDomain]; ok {
				podInfo.DomainName = domainName
			}

			nodePro.Projects = append(nodePro.Projects, podInfo)
			nodePro.PodCount++
			if nodePro.NodeIP == "" {
				nodePro.NodeIP = pod.Status.HostIP
			}
		}

		if nodePro.PodCount > 0 {
			nodePro.ClusterName = cluster.Name
			break
		}
	}

	sort.Slice(nodePro.Projects, func(i, j int) bool {
		return nodePro.Projects[i].AppName < nodePro.Projects[j].AppName
	})

	c.IndentedJSON(http.StatusOK, nodePro)
}

// GetPod ...
func (m *APIManager) GetPod(c *gin.Context) {
	appName := c.Param("appName")
	clusterName := c.Param("name")
	clusters := m.K8sMgr.GetAll(clusterName)
	ctx := context.Background()
	clusterPods := make([]*model.PodOfCluster, 0, 4)

	listOptions := &client.ListOptions{}
	listOptions.MatchingLabels(map[string]string{
		"app": appName,
	})
	for _, cluster := range clusters {
		pods := make([]*model.Pod, 0, 4)
		podList := &corev1.PodList{}
		err := cluster.Client.List(ctx, listOptions, podList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Error(err, "failed to get pods")
			AbortHTTPError(c, GetPodError, "", err)
			return
		}

		for i := range podList.Items {
			pod := &podList.Items[i]
			apiPod := &model.Pod{
				Name:            pod.GetName(),
				NodeIP:          pod.Status.HostIP,
				PodIP:           pod.Status.PodIP,
				ImageVersion:    "",
				StartTime:       pod.Status.StartTime.String(),
				ContainerStatus: nil,
			}
			apiPod.ContainerStatus = append(apiPod.ContainerStatus, &model.ContainerStatus{
				Name:         pod.Status.ContainerStatuses[0].Name,
				Ready:        pod.Status.ContainerStatuses[0].Ready,
				RestartCount: pod.Status.ContainerStatuses[0].RestartCount,
				Image:        pod.Status.ContainerStatuses[0].Image,
				ContainerID:  pod.Status.ContainerStatuses[0].ContainerID,
			})
			pods = append(pods, apiPod)
		}

		ofCluster := &model.PodOfCluster{
			ClusterName: cluster.Name,
			Pods:        pods,
		}
		clusterPods = append(clusterPods, ofCluster)
	}

	c.IndentedJSON(http.StatusOK, clusterPods)
}

// GetPodEvent return pod event
func (m *APIManager) GetPodEvent(c *gin.Context) {
	clusterName := c.Param("name")
	podName := c.Param("podName")
	namespace := c.DefaultQuery("namespace", "")
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)
	clusters := m.K8sMgr.GetAll(clusterName)

	result := []*model.Event{}
	for _, cluster := range clusters {
		ctx := context.Background()
		listOptions := &client.ListOptions{
			Namespace: namespace,
			Raw:       &metav1.ListOptions{Limit: limit},
		}
		events := &corev1.EventList{}

		err := cluster.Cache.List(ctx, listOptions, events)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Error(err, "failed to get pod events")
			AbortHTTPError(c, GetPodEventError, "", err)
			return
		}

		for _, event := range events.Items {
			item := &model.Event{
				Cluster:    cluster.GetName(),
				Namespace:  event.GetNamespace(),
				ObjectName: event.InvolvedObject.Name,
				ObjectKind: event.InvolvedObject.Kind,
				Type:       event.Type,
				Count:      event.Count,
				FirstTime:  event.FirstTimestamp,
				LastTime:   event.LastTimestamp,
				Message:    event.Message,
				Reason:     event.Reason,
			}
			if podName == "all" {
				result = append(result, item)
			} else if podName != "all" && item.ObjectName == podName {
				result = append(result, item)
			}
		}
	}

	c.IndentedJSON(http.StatusOK, result)
}

// DeletePodByName ...
func (m *APIManager) DeletePodByName(c *gin.Context) {
	clusterName := c.Param("name")
	podName := c.Param("podName")
	namespace := c.DefaultQuery("namespace", "default")

	cluster, err := m.K8sMgr.Get(clusterName)
	if err != nil {
		klog.Errorf("get cluster error: %v", err)
		AbortHTTPError(c, GetClusterError, "", err)
		return
	}

	ctx := context.Background()
	pod := &corev1.Pod{}
	err = cluster.Client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      podName,
	}, pod)
	if err != nil {
		klog.Errorf("get pod error: %v", err)
		AbortHTTPError(c, GetPodError, "", err)
		return
	}

	err = cluster.Client.Delete(ctx, pod)
	if err != nil {
		klog.Errorf("delete pod error: %v", err)
		AbortHTTPError(c, DeletePodError, "", err)
	}
	c.Status(http.StatusOK)
}

// DeletePodByGroup ...
func (m *APIManager) DeletePodByGroup(c *gin.Context) {
	clusterName := c.Param("name")
	appName := c.Param("appName")
	namespace := c.DefaultQuery("namespace", "default")
	group, ok := c.GetQuery("group")
	if !ok {
		AbortHTTPError(c, GetPodNotGroup, "no group label", nil)
		return
	}

	clusters := m.K8sMgr.GetAll(clusterName)
	ctx := context.Background()
	listOptions := &client.ListOptions{Namespace: namespace}
	listOptions.MatchingLabels(map[string]string{
		"app":       appName,
		"sym-group": group,
	})
	errorPods := []*corev1.Pod{}
	for _, cluster := range clusters {
		podList := &corev1.PodList{}
		err := cluster.Client.List(ctx, listOptions, podList)
		if err != nil {
			klog.Errorf("get pods error: %v", err)
			AbortHTTPError(c, GetPodError, "", err)
			return
		}

		for _, pod := range podList.Items {
			err = cluster.Client.Delete(ctx, &pod)
			if err != nil {
				klog.Errorf("delete pod error: %v", err)
				errorPods = append(errorPods, &pod)
			}
		}

	}
	c.JSON(http.StatusOK, gin.H{"errorPods": errorPods})
}
