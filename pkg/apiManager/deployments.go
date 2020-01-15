package apiManager

import (
	"context"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	appv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetDeployments get all deployments in assigned namespace
func (m *APIManager) GetDeployments(c *gin.Context) {
	clusterName := c.Param("name")
	appName := c.Param("appName")
	namespace := c.DefaultQuery("namespace", "")

	clusters := m.K8sMgr.GetAll(clusterName)

	ctx := context.Background()
	listOptions := &client.ListOptions{Namespace: namespace}
	listOptions.MatchingLabels(map[string]string{
		"app": appName,
	})
	result := []*model.DeploymentInfo{}
	for _, cluster := range clusters {
		deployments := &appv1.DeploymentList{}
		err := cluster.Client.List(ctx, listOptions, deployments)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Errorf("failed to get %s deployments: %v", cluster.GetName(), err)
			AbortHTTPError(c, GetDeploymentError, "", err)
			return
		}

		for _, deployment := range deployments.Items {
			info := model.DeploymentInfo{
				Name:                deployment.GetName(),
				Cluster:             cluster.GetName(),
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

	c.IndentedJSON(http.StatusOK, result)
}
