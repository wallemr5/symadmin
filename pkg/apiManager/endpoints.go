package apiManager

import (
	"context"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetEndpoints ...
func (m *APIManager) GetEndpoints(c *gin.Context) {
	clusterName := c.Param("name")
	appName := c.Param("appName")
	group := c.DefaultQuery("group", "")
	clusters := m.K8sMgr.GetAll(clusterName)

	ctx := context.Background()
	endpointsOfCluster := make([]*model.EndpointsOfCluster, 0, 4)
	eps := make([]*model.Endpoint, 0, 4)
	listOptions := &client.ListOptions{}
	listOptions.MatchingLabels(map[string]string{
		"app":   appName + "-svc",
		"group": group,
	})
	for _, cluster := range clusters {
		endpointList := &corev1.EndpointsList{}
		err := cluster.Client.List(ctx, listOptions, endpointList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Error(err, "failed to get endpoints")
			AbortHTTPError(c, GetEndpointError, "", err)
			return
		}

		for i := range endpointList.Items {
			ep := &endpointList.Items[i]

			for _, ss := range ep.Subsets {
				for _, addr := range ss.Addresses {
					eps = append(eps, &model.Endpoint{
						Subsets:           addr.IP,
						Name:              ep.Name,
						Namespace:         ep.Namespace,
						CreationTimestamp: ep.ObjectMeta.CreationTimestamp.Time.String(),
						Release:           "",
						ClusterName:       ep.ClusterName,
					})
				}
			}

		}
		ofCluster := model.EndpointsOfCluster{
			ClusterName: cluster.Name,
			Endpoint:    eps,
		}
		endpointsOfCluster = append(endpointsOfCluster, &ofCluster)
	}
	sort.Slice(endpointsOfCluster, func(i, j int) bool {
		return endpointsOfCluster[i].ClusterName < endpointsOfCluster[j].ClusterName
	})

	c.IndentedJSON(http.StatusOK, endpointsOfCluster)
}
