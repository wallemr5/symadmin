package v2

import (
	"context"
	"net/http"
	"regexp"
	"sort"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apimanager/model"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	deployments, err := m.getDeployments(clusterName, namespace, appName, group, zone, ldcLabel)
	if err != nil {
		klog.Errorf("failed to get deployments: %v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}

	var desired, updated, ready, available, unavailable int32
	if len(deployments) == 0 {
		stas, err := m.getStatefulset(clusterName, namespace, appName, group, zone, ldcLabel)
		if err != nil {
			klog.Errorf("failed to get statefulsets: %v", err)
			c.IndentedJSON(http.StatusBadRequest, gin.H{
				"success":   false,
				"message":   err.Error(),
				"resultMap": nil,
			})
			return
		}
		for _, sta := range stas {
			desired += *sta.DesiredReplicas
			updated += sta.UpdatedReplicas
			ready += sta.ReadyReplicas
			available += sta.AvailableReplicas
			unavailable += sta.UnavailableReplicas
		}
	} else {
		for _, deployment := range deployments {
			desired += *deployment.DesiredReplicas
			updated += deployment.UpdatedReplicas
			ready += deployment.ReadyReplicas
			available += deployment.AvailableReplicas
			unavailable += deployment.UnavailableReplicas
		}
	}

	stat := model.DeploymentStatInfo{
		DesiredReplicas:     desired,
		UpdatedReplicas:     updated,
		ReadyReplicas:       ready,
		AvailableReplicas:   available,
		UnavailableReplicas: unavailable,
		OK:                  desired == available && unavailable == 0,
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
		"resultMap": gin.H{
			"data": result,
			"stat": stat,
		},
	})
}

// GetAppGroupVersion ...
func (m *Manager) GetAppGroupVersion(c *gin.Context) {
	clusterName := c.Param("clusterCode")
	appName, ok := c.GetQuery("appName")
	namespace := c.DefaultQuery("namespace", "")

	if !ok || appName == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   "no appName",
			"resultMap": nil,
		})
		return
	}

	cluster, err := m.ClustersMgr.Get(clusterName)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}
	appset := &workloadv1beta1.AppSet{}
	err = cluster.Client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      appName,
	}, appset)

	type Result struct {
		Zone           string `json:"zone"`
		Group          string `json:"group"`
		Version        string `json:"version"`
		LastUpdateTime string `json:"lastUpdateTime"`
		OK             bool   `json:"ok"`
	}

	var result []*Result
	for _, cls := range appset.Spec.ClusterTopology.Clusters {
		for _, podSpec := range cls.PodSets {
			if podSpec.Mata == nil {
				continue
			}

			r := &Result{
				Zone:    podSpec.Mata["sym-zone"],
				Group:   podSpec.Mata["sym-group"],
				Version: podSpec.Version,
			}

			for _, cls := range m.ClustersMgr.GetAll() {
				deploy := &appsv1.Deployment{}
				err := cls.Client.Get(context.Background(), types.NamespacedName{
					Namespace: namespace,
					Name:      podSpec.Name,
				}, deploy)

				if err != nil {
					if apierrors.IsNotFound(err) {
						continue
					}
					klog.Errorf("get deployment error: %v", err)
					continue
				}

				r.LastUpdateTime = utils.FormatTime(deploy.Status.Conditions[0].LastUpdateTime.String())
				r.OK = deploy.Status.AvailableReplicas == *deploy.Spec.Replicas &&
					deploy.Status.UnavailableReplicas == 0
			}
			result = append(result, r)
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
// 2. 停止中 -> Terminating -> pod.DeletionTimestamp != nil
// 3. 已在线 -> Running
// 4. 发布失败 -> Failed
func (m *Manager) getPodListByAppName(clusterName, namespace, appName, group, zone, ldcLabel, podIP, phase string) ([]*model.Pod, error) {
	Terminating := "Terminating"

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
	if len(phase) > 0 && phase != Terminating {
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
		if phase == Terminating && pod.DeletionTimestamp == nil {
			continue
		}

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

		if pod.DeletionTimestamp != nil {
			apiPod.Phase = corev1.PodPhase(Terminating)
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

	cluster, err := m.ClustersMgr.Get(clusterName)
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

func (m *Manager) getDeployments(clusterName, namespace, appName, group, zone, ldcLabel string) ([]*model.DeploymentInfo, error) {
	result := make([]*model.DeploymentInfo, 0)

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
	deployments, err := m.Cluster.GetDeployment(listOptions, clusterName)
	if err != nil {
		klog.Errorf("failed to get deployments: %v", err)
		return nil, err
	}

	for _, deploy := range deployments {
		info := model.DeploymentInfo{
			Name:                deploy.Name,
			ClusterCode:         deploy.Labels["sym-cluster-info"],
			Annotations:         deploy.Annotations,
			Labels:              deploy.Labels,
			StartTime:           deploy.CreationTimestamp.Format("2006-01-02 15:04:05"),
			NameSpace:           deploy.Namespace,
			DesiredReplicas:     deploy.Spec.Replicas,
			UpdatedReplicas:     deploy.Status.UpdatedReplicas,
			ReadyReplicas:       deploy.Status.ReadyReplicas,
			AvailableReplicas:   deploy.Status.AvailableReplicas,
			UnavailableReplicas: deploy.Status.UnavailableReplicas,
			Group:               deploy.Labels["sym-group"],
			Selector:            deploy.Spec.Selector,
		}
		result = append(result, &info)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (m *Manager) getStatefulset(clusterName, namespace, appName, group, zone, ldcLabel string) ([]*model.DeploymentInfo, error) {
	result := make([]*model.DeploymentInfo, 0)

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

	listOptions := &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: options.AsSelector(),
	}
	stas, err := m.Cluster.GetStatefulsets(listOptions, clusterName)
	if err != nil {
		klog.Errorf("failed to get statefulsets: %v", err)
		return nil, err
	}

	for _, sta := range stas {
		info := model.DeploymentInfo{
			Name:                sta.Name,
			ClusterCode:         sta.Labels["sym-cluster-info"],
			Annotations:         sta.Annotations,
			Labels:              sta.Labels,
			StartTime:           sta.CreationTimestamp.Format("2006-01-02 15:04:05"),
			NameSpace:           sta.Namespace,
			DesiredReplicas:     sta.Spec.Replicas,
			UpdatedReplicas:     sta.Status.UpdatedReplicas,
			ReadyReplicas:       sta.Status.ReadyReplicas,
			AvailableReplicas:   sta.Status.ReadyReplicas,
			UnavailableReplicas: 0,
			Group:               sta.Labels["sym-group"],
			Selector:            sta.Spec.Selector,
		}
		result = append(result, &info)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}
