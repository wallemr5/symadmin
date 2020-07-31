package apiManager

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPodByLabels ...
func (m *APIManager) GetPodByLabels(c *gin.Context) {
	clusterName := c.Param("name")
	appName, ok := c.GetQuery("appName")
	group := c.DefaultQuery("group", "")
	ldcLabel := c.DefaultQuery("ldcLabel", "")
	namespace := c.DefaultQuery("namespace", "")
	zone := c.DefaultQuery("symZone", "")

	if !ok || appName == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   "no appName",
			"resultMap": nil,
		})
		return
	}

	result, err := m.getPodListByAppName(clusterName, namespace, appName, group, zone, ldcLabel)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   nil,
		"resultMap": result,
	})
}

// GetPodEvent return pod event
func (m *APIManager) GetPodEvent(c *gin.Context) {
	clusterName := c.Param("name")
	podName := c.Param("podName")
	namespace := c.Param("namespace")
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
			AbortHTTPError(c, GetPodEventError, "", err)
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
				FirstTime:   formatTime(event.FirstTimestamp.String()),
				LastTime:    formatTime(event.LastTimestamp.String()),
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

// GetPodByIP ...
func (m *APIManager) GetPodByIP(c *gin.Context) {
	clusterName := c.Param("name")
	podIP := c.Param("podIP")

	ctx := context.Background()
	list := &corev1.PodList{}

	clusters := m.K8sMgr.GetAll(clusterName)
	for _, cluster := range clusters {
		err := cluster.Client.List(ctx, list, &client.ListOptions{
			FieldSelector: fields.SelectorFromSet(fields.Set{"status.podIP": podIP})},
		)
		if err != nil {
			klog.Errorf("get pod from podIP error: %v", err)
		}
		if len(list.Items) > 0 {
			break
		}
	}
	if len(list.Items) == 0 {
		c.IndentedJSON(http.StatusOK, gin.H{
			"success":   false,
			"message":   "Can't found pod by this IP",
			"resultMap": nil,
		})
		return
	}
	pod := list.Items[0]
	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
		"resultMap": gin.H{
			"appName": pod.Labels["app"],
			"podName": pod.Name,
		},
	})
}

// DeletePodByName ...
func (m *APIManager) DeletePodByName(c *gin.Context) {
	clusterName := c.Param("name")
	podName := c.Param("podName")
	namespace := c.Param("namespace")

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
	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
	})
}

// GetPodByName ...
func (m *APIManager) GetPodByName(c *gin.Context) {
	clusterName := c.Param("name")
	podName := c.Param("podName")
	namespace := c.Param("namespace")

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

	apiPod := &model.Pod{
		Name:         pod.GetName(),
		Namespace:    pod.Namespace,
		ClusterCode:  cluster.GetName(),
		HostIP:       pod.Status.HostIP,
		PodIP:        pod.Status.PodIP,
		ImageVersion: "",
		StartTime:    formatTime(pod.Status.StartTime.String()),
		Containers:   nil,
		Labels:       pod.GetLabels(),
	}
	apiPod.Containers = append(apiPod.Containers, &model.ContainerStatus{
		Name:         pod.Status.ContainerStatuses[0].Name,
		Ready:        pod.Status.ContainerStatuses[0].Ready,
		RestartCount: pod.Status.ContainerStatuses[0].RestartCount,
		Image:        pod.Status.ContainerStatuses[0].Image,
		ContainerID:  pod.Status.ContainerStatuses[0].ContainerID,
	})

	c.IndentedJSON(http.StatusOK, apiPod)
}

// DeletePodByGroup ...
func (m *APIManager) DeletePodByGroup(c *gin.Context) {
	clusterName := c.Param("name")
	appName := c.Param("appName")
	namespace := c.Param("namespace")
	group, ok := c.GetPostForm("group")
	zone, _ := c.GetPostForm("symZone")
	ldcLabel, _ := c.GetPostForm("ldcLabel")
	if !ok {
		AbortHTTPError(c, GetPodNotGroup, "no group label", nil)
		return
	}

	clusters := m.K8sMgr.GetAll(clusterName)
	ctx := context.Background()
	options := labels.Set{}
	if group != "" {
		options["sym-group"] = group
	}
	if zone != "" {
		options["sym-zone"] = zone
	}
	if ldcLabel != "" {
		options["sym-ldc"] = ldcLabel
	}
	if appName != "all" {
		options["app"] = appName
	}
	listOptions := &client.ListOptions{Namespace: namespace, LabelSelector: options.AsSelector()}

	errorPods := []*corev1.Pod{}
	for _, cluster := range clusters {
		podList := &corev1.PodList{}
		err := cluster.Client.List(ctx, podList, listOptions)
		if err != nil {
			klog.Errorf("get pods error: %v", err)
			AbortHTTPError(c, GetPodError, "", err)
			return
		}

		for i := range podList.Items {
			pod := &podList.Items[i]
			err = cluster.Client.Delete(ctx, pod)
			if err != nil {
				klog.Errorf("delete pod error: %v", err)
				errorPods = append(errorPods, pod)
			}
		}

	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   nil,
		"errorPods": errorPods,
	})
}

// return Pod list， not PodOfCluster
func (m *APIManager) getPodListByAppName(clusterName, namespace, appName, group, zone, ldcLabel string) (map[string][]*model.Pod, error) {
	bluePods := make([]*model.Pod, 0, 4)
	greenPods := make([]*model.Pod, 0, 4)
	canaryPods := make([]*model.Pod, 0, 4)
	result := make(map[string][]*model.Pod)

	options := labels.Set{}
	if group != "" {
		options["sym-group"] = group
	}
	if zone != "" {
		options["sym-zone"] = zone
	}
	if ldcLabel != "" {
		options["sym-ldc"] = ldcLabel
	}
	if appName != "all" {
		options["app"] = appName
	}

	listOptions := &client.ListOptions{Namespace: namespace, LabelSelector: options.AsSelector()}
	podList, err := m.Cluster.GetPods(listOptions, clusterName)
	if err != nil {
		return nil, err
	}

	lb := labels.Set{
		"app": appName + "-svc",
	}
	endpointsListOptions := &client.ListOptions{Namespace: namespace, LabelSelector: lb.AsSelector()}
	endpointsList, err := m.Cluster.GetEndpoints(endpointsListOptions, clusterName)
	if err != nil {
		return nil, err
	}

	for _, pod := range podList {
		apiPod := &model.Pod{
			Name:         pod.GetName(),
			Namespace:    pod.Namespace,
			ClusterCode:  pod.GetLabels()["sym-cluster-info"],
			Annotations:  pod.GetAnnotations(),
			HostIP:       pod.Status.HostIP,
			Group:        pod.GetLabels()["sym-group"],
			PodIP:        pod.Status.PodIP,
			Phase:        pod.Status.Phase,
			ImageVersion: pod.GetAnnotations()["buildNumber_0"],
			CommitID:     pod.GetAnnotations()["gitCommit_0"],
			Containers:   nil,
			Labels:       pod.GetLabels(),
		}

		if pod.Status.StartTime != nil {
			apiPod.StartTime = formatTime(pod.Status.StartTime.String())
		}

		apiPod.Endpoints = false
		for _, ep := range endpointsList {
			for _, ss := range ep.Subsets {
				for _, addr := range ss.Addresses {
					if addr.TargetRef.Name == apiPod.Name {
						apiPod.Endpoints = true
						break
					}
				}
			}
		}

		for _, containerStatus := range pod.Status.ContainerStatuses {
			apiPod.RestartCount += containerStatus.RestartCount
			c := &model.ContainerStatus{
				Name:         containerStatus.Name,
				Ready:        containerStatus.Ready,
				RestartCount: containerStatus.RestartCount,
				Image:        containerStatus.Image,
				ContainerID:  containerStatus.ContainerID,
			}
			if containerStatus.LastTerminationState.Terminated != nil {
				apiPod.HasLastState = true
				t := containerStatus.LastTerminationState.Terminated
				c.LastState = &model.ContainerStateTerminated{
					ExitCode:    t.ExitCode,
					Signal:      t.Signal,
					Reason:      t.Reason,
					Message:     t.Message,
					StartedAt:   formatTime(t.StartedAt.String()),
					FinishedAt:  formatTime(t.FinishedAt.String()),
					ContainerID: t.ContainerID,
				}
			}
			apiPod.Containers = append(apiPod.Containers, c)
		}
		switch apiPod.Group {
		case "blue":
			bluePods = append(bluePods, apiPod)
		case "green":
			greenPods = append(greenPods, apiPod)
		case "gray":
			canaryPods = append(canaryPods, apiPod)
		}
	}

	sort.Slice(bluePods, func(i, j int) bool {
		return bluePods[i].Name < bluePods[j].Name
	})
	sort.Slice(greenPods, func(i, j int) bool {
		return greenPods[i].Name < greenPods[j].Name
	})
	sort.Slice(canaryPods, func(i, j int) bool {
		return canaryPods[i].Name < canaryPods[j].Name
	})

	result["blueGroup"] = bluePods
	result["greenGroup"] = greenPods
	result["canaryGroup"] = canaryPods

	return result, nil
}

func formatTime(dt string) string {
	loc, _ := time.LoadLocation("Asia/Chongqing")
	result, err := time.ParseInLocation("2006-01-02 15:04:05 -0700 MST", dt, loc)
	if err != nil {
		klog.Errorf("time parse error: %v", err)
		return ""
	}
	return result.Format("2006-01-02 15:04:05")
}
