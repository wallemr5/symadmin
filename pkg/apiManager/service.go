package apiManager

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetServices ...
func (m *APIManager) GetServices(c *gin.Context) {
	appName := c.Param("appName")
	clusterName := c.Param("name")
	clusters := m.K8sMgr.GetAll(clusterName)
	ctx := context.Background()
	svcResult := make([]model.ServiceInfo, 0, 4)
	listOptions := &client.ListOptions{}
	listOptions.MatchingLabels(map[string]string{
		"app": appName + "-svc", // 协商service selector 需要加"-svc"加后缀
	})
	for _, cluster := range clusters {
		svclist := &corev1.ServiceList{}
		err := cluster.Client.List(ctx, listOptions, svclist)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Error(err, "failed to get nodes")
			AbortHTTPError(c, GetServiceError, "", err)
			return
		}
		for i := range svclist.Items {
			service := &svclist.Items[i]
			serviceSpec := service.Spec

			info := model.ServiceInfo{
				ClusterName: cluster.Name,
				NameSpace:   service.Namespace,
				ClusterIP:   serviceSpec.ClusterIP,
				Type:        string(serviceSpec.Type),
				Ports:       serviceSpec.Ports,
				Selector:    serviceSpec.Selector,
			}

			svcResult = append(svcResult, info)
		}
	}

	c.IndentedJSON(http.StatusOK, svcResult)
}
