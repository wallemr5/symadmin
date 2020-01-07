package apiManager

import (
	"context"
	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (m *ApiManager) GetEndpoints(c *gin.Context) {
	// clusterName := c.Param("name")

	endpointName := c.Param("endpointName")

	clusters := m.K8sMgr.GetAll()

	ctx := context.Background()
	eps := make([]*model.Endpoints, 0, 4)
	//pods := make([]*model.Pod, 0, 4)
	listOptions := &client.ListOptions{}
	for _, cluster := range clusters {
		if cluster.Status == k8smanager.ClusterOffline {
			continue
		}

		endpointList := &corev1.EndpointsList{}
		//podList := &corev1.PodList{}
		err := cluster.Client.List(ctx, listOptions, endpointList)
		if err != nil {

			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Error(err, "failed to get endpoints")
			break
		}

		for i := range endpointList.Items {
			ep := &endpointList.Items[i]
			if ep.Name == endpointName {
				//fmt.Println(ep)
				for _, ss := range ep.Subsets {
					for _, addr := range ss.Addresses {
						eps = append(eps, &model.Endpoints{
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
		}
		c.JSON(http.StatusOK, eps)
	}
}