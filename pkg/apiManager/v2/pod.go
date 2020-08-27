package v2

import (
	"context"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPodByLabels ...
func (m *Manager) GetPodByLabels(c *gin.Context) {
	clusterName := c.Param("clusterCode")
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

	result, err := m.getPodListByAppName(clusterName, namespace, appName, group, zone, ldcLabel, podIP, phase)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
		"resultMap": gin.H{
			"data": result,
		},
	})
}

// GetAppGroupVersion ...
func (m *Manager) GetAppGroupVersion(c *gin.Context) {
	clusterName := c.Param("clusterCode")
	appName, ok := c.GetQuery("appName")
	group := c.DefaultQuery("group", "")
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

	pods, err := m.getPodListByAppName(clusterName, namespace, appName, group, zone, "", "", "")
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}

	result := make(map[string]string)
	for _, pod := range pods {
		_, ok := result[pod.Group]
		if !ok {
			result[pod.Group] = pod.ImageVersion
		}
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   nil,
		"resultMap": result,
	})
}

// TODO phase:
// 1. 发布中 -> Pending
// 2. 回滚中 -> Pending / ContainerCreating
// 3. 停止中 -> Terminating -> pod.DeletionTimestamp != nil
// 4. 准备中 -> ContainerCreating -> pod.status.phase = Pending & container.state = Waiting ,container.state.reason = ContainerCreating
// 5. 已在线 -> Running
// 6. 发布失败 -> Failed
func (m *Manager) getPodListByAppName(clusterName, namespace, appName, group, zone, ldcLabel, podIP, phase string) ([]*model.Pod, error) {
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

	fieldMap := make(map[string]string)
	if len(podIP) > 0 {
		fieldMap["status.podIP"] = podIP
	}
	if len(phase) > 0 {
		fieldMap["status.phase"] = phase
	}
	if len(fieldMap) > 0 {
		set := fields.Set(fieldMap)
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
			Id:           getPodID(pod.GetName()),
			Name:         pod.GetName(),
			Namespace:    pod.Namespace,
			ClusterCode:  pod.GetLabels()["sym-cluster-info"],
			Annotations:  pod.GetAnnotations(),
			HostIP:       pod.Status.HostIP,
			Group:        pod.GetLabels()["sym-group"],
			Zone:         pod.GetLabels()["sym-zone"],
			PodIP:        pod.Status.PodIP,
			Phase:        pod.Status.Phase,
			ImageVersion: pod.GetAnnotations()["buildNumber_0"],
			CommitID:     pod.GetAnnotations()["gitCommit_0"],
			Containers:   nil,
			Labels:       pod.GetLabels(),
		}

		if pod.Status.StartTime != nil {
			apiPod.StartTime = utils.FormatTime(pod.Status.StartTime.String())
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
					StartedAt:   utils.FormatTime(t.StartedAt.String()),
					FinishedAt:  utils.FormatTime(t.FinishedAt.String()),
					ContainerID: t.ContainerID,
				}
			}
			apiPod.Containers = append(apiPod.Containers, c)
		}

		pods = append(pods, apiPod)
	}

	return pods, nil
}

// DeletePodByName ...
func (m *Manager) DeletePodByName(c *gin.Context) {
	clusterName := c.Param("clusterCode")
	podName := c.Param("podName")
	namespace := c.Param("namespace")

	cluster, err := m.K8sMgr.Get(clusterName)
	if err != nil {
		klog.Errorf("get cluster error: %v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
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
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}

	err = cluster.Client.Delete(ctx, pod)
	if err != nil {
		klog.Errorf("delete pod error: %v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   nil,
		"resultMap": nil,
	})
}

func getPodID(podName string) string {
	reg, err := regexp.Compile("(-[r|g]z[0-9]+[a-z])?(-green|-blue|-canary)-(.*)")
	if err != nil {
		return ""
	}
	submatch := reg.FindStringSubmatch(podName)
	if len(submatch) > 3 {
		return submatch[3]
	}
	return ""
}