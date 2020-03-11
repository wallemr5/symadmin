package apiManager

import (
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetDeployments get all deployments in assigned namespace
func (m *APIManager) GetDeployments(c *gin.Context) {
	clusterName := c.Param("name")
	appName := c.Param("appName")
	group := c.DefaultQuery("group", "")
	zone := c.DefaultQuery("symZone", "")
	ldcLabel := c.DefaultQuery("ldcLabel", "")
	namespace := c.DefaultQuery("namespace", "")

	result, err := m.getDeployments(clusterName, namespace, appName, group, zone, ldcLabel)
	if err != nil {
		klog.Errorf("failed to get deployments: %v", err)
		AbortHTTPError(c, GetDeploymentError, "", err)
		return
	}

	c.IndentedJSON(http.StatusOK, result)
}

// GetDeploymentsStat ...
func (m *APIManager) GetDeploymentsStat(c *gin.Context) {
	clusterName := c.Param("name")
	appName := c.DefaultQuery("appName", "all")
	group := c.DefaultQuery("group", "")
	ldcLabel := c.DefaultQuery("ldcLabel", "")
	zone := c.DefaultQuery("symZone", "")
	namespace := c.DefaultQuery("namespace", "")

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
	for _, deployment := range deployments {
		desired += *deployment.DesiredReplicas
		updated += deployment.UpdatedReplicas
		ready += deployment.ReadyReplicas
		available += deployment.AvailableReplicas
		unavailable += deployment.UnavailableReplicas
	}

	result := model.DeploymentStatInfo{
		DesiredReplicas:     desired,
		UpdatedReplicas:     updated,
		ReadyReplicas:       ready,
		AvailableReplicas:   available,
		UnavailableReplicas: unavailable,
		OK:                  desired == available && unavailable == 0,
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   nil,
		"resultMap": result,
	})
}

func (m *APIManager) getDeployments(clusterName, namespace, appName, group, zone, ldcLabel string) ([]*model.DeploymentInfo, error) {
	result := make([]*model.DeploymentInfo, 0)

	listOptions := &client.ListOptions{Namespace: namespace}
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

	listOptions.MatchingLabels(options)
	deployments, err := m.Cluster.GetDeployment(listOptions, clusterName)
	if err != nil {
		klog.Errorf("failed to get deployments: %v", err)
		return nil, err
	}

	for _, deploy := range deployments {
		info := model.DeploymentInfo{
			Name:                deploy.GetName(),
			ClusterCode:         deploy.GetClusterName(),
			Annotations:         deploy.GetAnnotations(),
			Labels:              deploy.GetLabels(),
			StartTime:           deploy.GetCreationTimestamp().Format("2006-01-02 15:04:05"),
			NameSpace:           deploy.GetNamespace(),
			DesiredReplicas:     deploy.Spec.Replicas,
			UpdatedReplicas:     deploy.Status.UpdatedReplicas,
			ReadyReplicas:       deploy.Status.ReadyReplicas,
			AvailableReplicas:   deploy.Status.AvailableReplicas,
			UnavailableReplicas: deploy.Status.UnavailableReplicas,
			Group:               deploy.GetLabels()["sym-group"],
			Selector:            deploy.Spec.Selector,
		}
		result = append(result, &info)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (m *APIManager) getDeployments(clusterName, namespace, appName, group, zone, ldcLabel string) ([]*model.DeploymentInfo, error) {
	result := make([]*model.DeploymentInfo, 0)

	listOptions := &client.ListOptions{Namespace: namespace}
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

	listOptions.MatchingLabels(options)
	deployments, err := m.Cluster.GetDeployment(listOptions, clusterName)
	if err != nil {
		klog.Errorf("failed to get deployments: %v", err)
		return nil, err
	}

	for _, deploy := range deployments {
		info := model.DeploymentInfo{
			Name:                deploy.GetName(),
			ClusterCode:         deploy.ClusterName,
			Annotations:         deploy.GetAnnotations(),
			Labels:              deploy.GetLabels(),
			StartTime:           deploy.GetCreationTimestamp().Format("2006-01-02 15:04:05"),
			NameSpace:           deploy.GetNamespace(),
			DesiredReplicas:     deploy.Spec.Replicas,
			UpdatedReplicas:     deploy.Status.UpdatedReplicas,
			ReadyReplicas:       deploy.Status.ReadyReplicas,
			AvailableReplicas:   deploy.Status.AvailableReplicas,
			UnavailableReplicas: deploy.Status.UnavailableReplicas,
			Group:               deploy.GetLabels()["sym-group"],
			Selector:            deploy.Spec.Selector,
		}
		result = append(result, &info)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}
