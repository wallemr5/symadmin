package apiManager

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// GetDeployments get all deployments in assigned namespace
func (m *ApiManager) GetDeployments(c *gin.Context) {
	clusterName := c.Param("name")
	namespace := c.DefaultQuery("namespace", "default")

	cluster, err := m.K8sMgr.Get(clusterName)
	if err != nil {
		klog.Errorf("get cluster error: %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "can not get cluster."})
		return
	}
	if cluster.Status == k8smanager.ClusterOffline {
		c.JSON(http.StatusBadRequest, gin.H{"error": "the cluster you get is offline."})
		return
	}

	deployments, err := cluster.KubeCli.AppsV1().Deployments(namespace).
		List(metav1.ListOptions{})
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
