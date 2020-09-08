package v1

import (
	"context"
	"net/http"
	"sort"

	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apimanager/model"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetServices ...
func (m *Manager) GetServices(c *gin.Context) {
	appName := c.Param("appName")
	clusterName := c.Param("name")
	namespace := c.Param("namespace")
	clusters := m.ClustersMgr.GetAll(clusterName)

	result, err := getService(clusters, namespace, appName)
	if err != nil {
		klog.Error(err, "failed to get service")
		AbortHTTPError(c, GetServiceError, "", err)
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   nil,
		"resultMap": result,
	})
}

// GetServiceInfo ...
func (m *Manager) GetServiceInfo(c *gin.Context) {
	clusterName := c.Param("name")
	namespace := c.Param("namespace")
	serviceName := c.Param("svcName")
	outFormat := c.DefaultQuery("format", "yaml")

	cluster, err := m.ClustersMgr.Get(clusterName)
	if err != nil {
		klog.Errorf("get cluster error: %v", err)
		AbortHTTPError(c, GetClusterError, "", err)
		return
	}

	ctx := context.Background()
	service := &corev1.Service{}
	err = cluster.Client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      serviceName,
	}, service)
	if err != nil {
		klog.Errorf("get service error: %v", err)
		AbortHTTPError(c, GetServiceError, "", err)
		return
	}

	if outFormat == "yaml" {
		svcByte, err := yaml.Marshal(service)
		if err != nil {
			klog.Errorf("Marshal service info err:%+v", err)
			AbortHTTPError(c, 0, "", err)
			return
		}

		c.IndentedJSON(http.StatusOK, gin.H{
			"success":   true,
			"message":   nil,
			"resultMap": string(svcByte),
		})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   nil,
		"resultMap": service,
	})

}

func getService(clusters []*k8smanager.Cluster, namespace, appName string) ([]*model.ServiceInfo, error) {
	ctx := context.Background()
	result := make([]*model.ServiceInfo, 0, 4)
	lb := labels.Set{
		"app": appName + "-svc",
	}
	options := &client.ListOptions{Namespace: namespace, LabelSelector: lb.AsSelector()}

	for _, cluster := range clusters {
		svclist := &corev1.ServiceList{}
		err := cluster.Client.List(ctx, svclist, options)
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
