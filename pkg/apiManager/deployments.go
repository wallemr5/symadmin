package apiManager

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	appv1 "k8s.io/api/apps/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetDeployments get all deployments in assigned namespace
func (m *APIManager) GetDeployments(c *gin.Context) {
	clusterName := c.Param("name")
	namespace := c.DefaultQuery("namespace", "default")

	cluster, err := m.K8sMgr.Get(clusterName)
	if err != nil {
		klog.Errorf("get cluster error: %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "can not get cluster."})
		return
	}

	ctx := context.Background()
	listOptions := &client.ListOptions{Namespace: namespace}
	deployments := &appv1.DeploymentList{}

	err = cluster.Client.List(ctx, listOptions, deployments)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "get deployments error"})
		return
	}

	result := make([]*model.DeploymentInfo, 0)
	for _, deployment := range deployments.Items {
		info := model.DeploymentInfo{
			Name:                deployment.GetName(),
			Cluster:             deployment.GetClusterName(),
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

	c.JSON(http.StatusOK, gin.H{"deployments": result})
}
