package apiManager

import (
	"context"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
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
	group := c.DefaultQuery("group", "")

	result, err := getService(clusters, appName, group)
	if err != nil {
		klog.Error(err, "failed to get service")
		AbortHTTPError(c, GetServiceError, "", err)
		return
	}
	c.IndentedJSON(http.StatusOK, result)
}

func getService(clusters []*k8smanager.Cluster, appName, group string) ([]*model.ServiceInfo, error) {
	ctx := context.Background()
	result := make([]*model.ServiceInfo, 0, 4)
	options := &client.ListOptions{}
	options.MatchingLabels(map[string]string{
		"app":       appName + "-svc", // 协商service selector 需要加"-svc"加后缀
		"sym-group": group,
	})
	for _, cluster := range clusters {
		svclist := &corev1.ServiceList{}
		err := cluster.Client.List(ctx, options, svclist)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		for i := range svclist.Items {
			service := &svclist.Items[i]
			serviceSpec := service.Spec

			info := &model.ServiceInfo{
				Name:        service.Name,
				ClusterName: cluster.Name,
				NameSpace:   service.Namespace,
				ClusterIP:   serviceSpec.ClusterIP,
				Type:        string(serviceSpec.Type),
				Ports:       serviceSpec.Ports,
				Selector:    serviceSpec.Selector,
			}

			result = append(result, info)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ClusterName < result[j].ClusterName
	})
	return result, nil
}
