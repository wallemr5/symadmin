package apiManager

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
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

	var isFind bool
	var findProject *model.Project
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
			var appName string

			isFind = false
			pod := &podList.Items[i]
			if appName, ok = pod.GetLabels()[labels.ObserveMustLabelAppName]; !ok {
				continue
			}

			if _, ok = pod.GetLabels()[labels.ObserveMustLabelGroupName]; !ok {
				continue
			}

			for _, p := range nodePro.Projects {
				if p.AppName == appName {
					findProject = p
					isFind = true
					break
				}
			}

			if isFind && findProject != nil {
				nodePro.PodCount++
				findProject.PodCount++
				findProject.Instances = append(findProject.Instances, pod.Status.PodIP)
				continue
			}

			podInfo := &model.Project{}
			podInfo.AppName = appName
			podInfo.PodCount++
			podInfo.Instances = append(podInfo.Instances, pod.Status.PodIP)
			if domainName, ok := pod.GetLabels()[labels.ObserveMustLabelDomain]; ok {
				podInfo.DomainName = domainName
			} else {
				svcList := &corev1.ServiceList{}
				svcListOptions := &client.ListOptions{}
				svcListOptions.MatchingLabels(map[string]string{
					"app": appName + "-svc",
				})
				err := cluster.Client.List(ctx, svcListOptions, svcList)
				if err == nil && len(svcList.Items) > 0 {
					if domainName, ok := svcList.Items[0].GetLabels()[labels.ObserveMustLabelDomain]; ok {
						podInfo.DomainName = domainName
					}
				}
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
	group := c.DefaultQuery("group", "")
	clusterName := c.Param("name")
	clusters := m.K8sMgr.GetAll(clusterName)
	result, err := getPodByAppName(clusters, appName, group)
	if err != nil {
		klog.Error(err, "failed to get pods")
		AbortHTTPError(c, GetPodError, "", err)
		return
	}
	c.IndentedJSON(http.StatusOK, result)
}

// GetPodByLabels ...
func (m *APIManager) GetPodByLabels(c *gin.Context) {
	clusterName := c.Param("name")
	appName, ok := c.GetQuery("appName")
	group := c.DefaultQuery("group", "")
	ldcLabel := c.DefaultQuery("ldcLabel", "")
	namespace := c.DefaultQuery("namespace", "")
	zone := c.DefaultQuery("zone", "")
	clusters := m.K8sMgr.GetAll(clusterName)

	if !ok || appName == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   "no appName",
			"resultMap": nil,
		})
		return
	}

	result, err := getPodListByAppName(clusters, appName, group, ldcLabel, namespace, zone)
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
				ClusterName: cluster.GetName(),
				Namespace:   event.GetNamespace(),
				ObjectName:  event.InvolvedObject.Name,
				ObjectKind:  event.InvolvedObject.Kind,
				Type:        event.Type,
				Count:       event.Count,
				FirstTime:   event.FirstTimestamp,
				LastTime:    event.LastTimestamp,
				Message:     event.Message,
				Reason:      event.Reason,
			}
			if podName == "all" {
				result = append(result, item)
			} else if podName != "all" && item.ObjectName == podName {
				result = append(result, item)
			}
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastTime.Before(&result[j].LastTime)
	})

	c.IndentedJSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   nil,
		"resultMap": gin.H{"events": result},
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
	group, ok := c.GetQuery("group")
	zone := c.DefaultQuery("zone", "")
	ldcLabel := c.DefaultQuery("ldcLabel", "")
	if !ok {
		AbortHTTPError(c, GetPodNotGroup, "no group label", nil)
		return
	}

	clusters := m.K8sMgr.GetAll(clusterName)
	ctx := context.Background()
	options := make(map[string]string)
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
	listOptions := &client.ListOptions{Namespace: namespace}
	listOptions.MatchingLabels(options)
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

func getPodByAppName(clusters []*k8smanager.Cluster, appName, group string) ([]*model.PodOfCluster, error) {
	ctx := context.Background()
	clusterPods := make([]*model.PodOfCluster, 0, 4)
	listOptions := &client.ListOptions{}
	listOptions.MatchingLabels(map[string]string{
		"app":   appName,
		"group": group,
	})
	for _, cluster := range clusters {
		pods := make([]*model.Pod, 0, 4)
		podList := &corev1.PodList{}
		err := cluster.Client.List(ctx, listOptions, podList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}

		for i := range podList.Items {
			pod := &podList.Items[i]
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
			pods = append(pods, apiPod)
		}
		sort.Slice(pods, func(i, j int) bool {
			return pods[i].Name < pods[j].Name
		})

		ofCluster := &model.PodOfCluster{
			ClusterName: cluster.Name,
			Pods:        pods,
		}
		clusterPods = append(clusterPods, ofCluster)
	}
	sort.Slice(clusterPods, func(i, j int) bool {
		return clusterPods[i].ClusterName < clusterPods[j].ClusterName
	})
	return clusterPods, nil
}

// return Pod list， not PodOfCluster
func getPodListByAppName(clusters []*k8smanager.Cluster, appName, group, ldcLabel, namespace, zone string) (map[string][]*model.Pod, error) {
	ctx := context.Background()
	bluePods := make([]*model.Pod, 0, 4)
	greenPods := make([]*model.Pod, 0, 4)
	canaryPods := make([]*model.Pod, 0, 4)
	result := make(map[string][]*model.Pod)

	options := make(map[string]string)
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

	listOptions := &client.ListOptions{Namespace: namespace}
	listOptions.MatchingLabels(options)

	endpointsListOptions := &client.ListOptions{Namespace: namespace}
	endpointsListOptions.MatchingLabels(map[string]string{"app": appName + "-svc"})

	for _, cluster := range clusters {
		podList := &corev1.PodList{}
		err := cluster.Client.List(ctx, listOptions, podList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		// look up endpoint
		endpointList := &corev1.EndpointsList{}
		err = cluster.Client.List(ctx, endpointsListOptions, endpointList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}

		for _, pod := range podList.Items {
			apiPod := &model.Pod{
				Name:         pod.GetName(),
				Namespace:    pod.Namespace,
				ClusterCode:  cluster.GetName(),
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
			for _, ep := range endpointList.Items {
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
				apiPod.Containers = append(apiPod.Containers, &model.ContainerStatus{
					Name:         containerStatus.Name,
					Ready:        containerStatus.Ready,
					RestartCount: containerStatus.RestartCount,
					Image:        containerStatus.Image,
					ContainerID:  containerStatus.ContainerID,
					LastState:    containerStatus.LastTerminationState.Terminated,
				})
				if containerStatus.LastTerminationState.Terminated != nil {
					apiPod.HasLastState = true
				}
			}
			switch apiPod.Group {
			case "blue":
				bluePods = append(bluePods, apiPod)
			case "green":
				greenPods = append(greenPods, apiPod)
			case "canary":
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
	}
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
