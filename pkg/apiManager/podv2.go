package apiManager

import (
	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPodByLabels ...
func (m *APIManager) GetPodByLabelsV2(c *gin.Context) {
	clusterName := c.Param("name")
	appName, ok := c.GetQuery("appName")
	group := c.DefaultQuery("group", "")
	ldcLabel := c.DefaultQuery("ldcLabel", "")
	namespace := c.DefaultQuery("namespace", "")
	zone := c.DefaultQuery("symZone", "")
	podIP := c.DefaultQuery("podIP", "")
	phase := c.DefaultQuery("phase", "")

	if !ok || appName == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   "no appName",
			"resultMap": nil,
		})
		return
	}

	result, err := m.getPodListByAppNameV2(clusterName, namespace, appName, group, zone, ldcLabel, podIP, phase)
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

// return Pod list， not PodOfCluster
func (m *APIManager) getPodListByAppNameV2(clusterName, namespace, appName, group, zone, ldcLabel, podIP, phase string) ([]*model.Pod, error) {
	pods := make([]*model.Pod, 0, 4)
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
	options["app"] = appName

	listOptions := &client.ListOptions{Namespace: namespace, LabelSelector: options.AsSelector()}
	if len(podIP) > 0 {
		set := fields.Set{"status.podIP": podIP}

		listOptions.FieldSelector = set.AsSelector()
	}
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

		pods = append(pods, apiPod)
	}

	return pods, nil
}
