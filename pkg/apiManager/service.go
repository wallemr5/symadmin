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

func (m *ApiManager) GetServices(c *gin.Context) {
	appName := c.Param("appName")
	clusters := m.K8sMgr.GetAll()
	ctx := context.Background()
	svcResult := make([]model.ServiceInfo, 0, 4)
	listOptions := &client.ListOptions{
		LabelSelector: nil,
		FieldSelector: nil,
		Namespace:     "",
		Raw:           nil,
	}
	listOptions.MatchingLabels(map[string]string{
		"app": appName,
	})
	for _, cluster := range clusters {
		svclist := &corev1.ServiceList{}
		err := cluster.Client.List(ctx, listOptions, svclist)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Error(err, "failed to get nodes")
			break
		}
		for i := range svclist.Items {
			service := &svclist.Items[i]
			serviceSpec := service.Spec

			info := model.ServiceInfo{
				NameSpace: service.Namespace,
				ClusterIP: serviceSpec.ClusterIP,
				Type:      string(serviceSpec.Type),
				Ports:     serviceSpec.Ports,
				Selector:  serviceSpec.Selector,
			}

			svcResult = append(svcResult, info)
		}
	}

	c.IndentedJSON(http.StatusOK, svcResult)
}
