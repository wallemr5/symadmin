package apiManager

import (
	"context"
	"net/http"
	"sort"
	"strconv"

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
	clusters := m.K8sMgr.GetAll(clusterName)

	ctx := context.Background()
	eps := make([]*model.Endpoint, 0, 4)
	listOptions := &client.ListOptions{}
	listOptions.MatchingLabels(map[string]string{
		"app": appName + "-svc",
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
			c.IndentedJSON(http.StatusBadRequest, gin.H{
				"success":   false,
				"message":   err.Error(),
				"resultMap": nil,
			})
			return
		}

		for _, ep := range endpointList.Items {
			var subset []string
			for _, ss := range ep.Subsets {
				port := strconv.Itoa(int(ss.Ports[0].Port))
				for _, addr := range ss.Addresses {
					subset = append(subset, addr.IP+":"+port)
				}
			}
			eps = append(eps, &model.Endpoint{
				Subsets:           subset,
				Labels:            ep.GetLabels(),
				Name:              ep.Name,
				Namespace:         ep.Namespace,
				CreationTimestamp: ep.ObjectMeta.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
				ClusterCode:       ep.ClusterName,
			})

		}
	}
	sort.Slice(eps, func(i, j int) bool {
		return eps[i].ClusterCode < eps[j].ClusterCode
	})

	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
		"resultMap": gin.H{
			"endpoints": eps,
		},
	})
}
