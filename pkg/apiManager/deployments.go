package apiManager

import (
	"context"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	appv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetDeployments get all deployments in assigned namespace
func (m *APIManager) GetDeployments(c *gin.Context) {
	clusterName := c.Param("name")
	appName := c.Param("appName")
	group := c.DefaultQuery("group", "")
	ldcLabel := c.DefaultQuery("ldcLabel", "")
	namespace := c.DefaultQuery("namespace", "")
	clusters := m.K8sMgr.GetAll(clusterName)

	result, err := getDeployments(clusters, namespace, appName, group, ldcLabel)
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
	appName := c.Param("appName")
	group := c.DefaultQuery("group", "")
	ldcLabel := c.DefaultQuery("ldcLabel", "")
	namespace := c.DefaultQuery("namespace", "")
	clusters := m.K8sMgr.GetAll(clusterName)

	deployments, err := getDeployments(clusters, namespace, appName, group, ldcLabel)
	if err != nil {
		klog.Errorf("failed to get deployments: %v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
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
		"message":   "",
		"resultMap": result,
	})
}

func getDeployments(clusters []*k8smanager.Cluster, namespace, appName, group, ldcLabel string) ([]*model.DeploymentInfo, error) {
	ctx := context.Background()
	listOptions := &client.ListOptions{Namespace: namespace}
	options := make(map[string]string)
	if group != "" {
		options["sym-group"] = group
	}
	if ldcLabel != "" {
		options["sym-ldc"] = ldcLabel
	}
	if appName != "all" {
		options["app"] = appName
	}

	listOptions.MatchingLabels(options)
	result := []*model.DeploymentInfo{}
	for _, cluster := range clusters {
		deployments := &appv1.DeploymentList{}
		err := cluster.Client.List(ctx, listOptions, deployments)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}

		for _, deployment := range deployments.Items {
			info := model.DeploymentInfo{
				Name:                deployment.GetName(),
				ClusterName:         cluster.GetName(),
				NameSpace:           deployment.GetNamespace(),
				DesiredReplicas:     deployment.Spec.Replicas,
				UpdatedReplicas:     deployment.Status.UpdatedReplicas,
				ReadyReplicas:       deployment.Status.ReadyReplicas,
				AvailableReplicas:   deployment.Status.AvailableReplicas,
				UnavailableReplicas: deployment.Status.UnavailableReplicas,
				Group:               deployment.GetLabels()["sym-group"],
				Selector:            deployment.Spec.Selector,
				CreationTimestamp:   deployment.GetCreationTimestamp(),
			}
			result = append(result, &info)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}
