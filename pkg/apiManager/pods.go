package apiManager

import (
	"context"
	"net/http"
	"sort"
	"strconv"

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

	c.IndentedJSON(http.StatusOK, result)
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
	c.Status(http.StatusOK)
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
		Name:            pod.GetName(),
		Namespace:       pod.Namespace,
		ClusterName:     cluster.GetName(),
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

	c.IndentedJSON(http.StatusOK, apiPod)
}

// DeletePodByGroup ...
func (m *APIManager) DeletePodByGroup(c *gin.Context) {
	clusterName := c.Param("name")
	appName := c.Param("appName")
	namespace := c.Param("namespace")
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
				Name:            pod.GetName(),
				Namespace:       pod.Namespace,
				ClusterName:     cluster.GetName(),
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
func getPodListByAppName(clusters []*k8smanager.Cluster, appName, group string) ([]*model.Pod, error) {
	ctx := context.Background()
	pods := make([]*model.Pod, 0, 4)
	listOptions := &client.ListOptions{}
	listOptions.MatchingLabels(map[string]string{
		"app":       appName,
		"sym-group": group,
	})
	endpointsListOptions := &client.ListOptions{}
	endpointsListOptions.MatchingLabels(map[string]string{
		"app":       appName + "-svc",
		"sym-group": group,
	})
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

		for i := range podList.Items {
			pod := &podList.Items[i]
			apiPod := &model.Pod{
				Name:            pod.GetName(),
				Namespace:       pod.Namespace,
				ClusterName:     cluster.GetName(),
				NodeIP:          pod.Status.HostIP,
				PodIP:           pod.Status.PodIP,
				Phase:           pod.Status.Phase,
				ImageVersion:    "",
				StartTime:       pod.Status.StartTime.String(),
				ContainerStatus: nil,
			}

			apiPod.HasEndpoint = false
			for i := range endpointList.Items {
				ep := &endpointList.Items[i]
				for _, ss := range ep.Subsets {
					for _, addr := range ss.Addresses {
						if addr.TargetRef.Name == apiPod.Name {
							apiPod.HasEndpoint = true
							break
						}
					}
				}
			}

			for i := range pod.Status.ContainerStatuses {
				apiPod.RestartCount += pod.Status.ContainerStatuses[i].RestartCount
				apiPod.ContainerStatus = append(apiPod.ContainerStatus, &model.ContainerStatus{
					Name:         pod.Status.ContainerStatuses[i].Name,
					Ready:        pod.Status.ContainerStatuses[i].Ready,
					RestartCount: pod.Status.ContainerStatuses[i].RestartCount,
					Image:        pod.Status.ContainerStatuses[i].Image,
					ContainerID:  pod.Status.ContainerStatuses[i].ContainerID,
				})
			}
			pods = append(pods, apiPod)
		}

		sort.Slice(pods, func(i, j int) bool {
			return pods[i].Name < pods[j].Name
		})
	}

	return pods, nil
}
