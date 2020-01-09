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

// GetNodeInfo ...
func (m *APIManager) GetNodeInfo(c *gin.Context) {
	clusterName := c.Param("name")
	nodeName := c.Param("nodeName")
	clusters := m.K8sMgr.GetAll(clusterName)

	ctx := context.Background()
	nodes := make([]model.NodeInfo, 0, 4)
	listOptions := &client.ListOptions{
		LabelSelector: nil,
		FieldSelector: nil,
		Namespace:     "",
		Raw:           nil,
	}
	listOptions.MatchingLabels(map[string]string{
		"nodeName": nodeName,
	})

	for _, cluster := range clusters {
		nodeList := &corev1.NodeList{}
		err := cluster.Client.List(ctx, listOptions, nodeList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Error(err, "failed to get nodes")
			AbortHTTPError(c, GetNodeError, "", err)
			return
		}

		for i := range nodeList.Items {
			node := &nodeList.Items[i]
			cpu, _ := node.Status.Allocatable.Cpu().AsInt64()
			memory, _ := node.Status.Allocatable.Memory().AsInt64()
			memory = memory / 1024 / 1024 / 1024
			nodeInfo := model.NodeInfo{
				Name:          node.Name,
				HostIP:        node.Status.Addresses[0].Address,
				Status:        string(node.Status.Conditions[len(node.Status.Conditions)-1].Type),
				CPU:           cpu,
				KernelVersion: node.Status.NodeInfo.KernelVersion,
				Architecture:  node.Status.NodeInfo.Architecture,
				System:        node.Status.NodeInfo.OSImage,
				MemorySize:    memory,
				JoinDate:      node.CreationTimestamp.Format("2006-01-02"),
				DockerVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
			}
			nodes = append(nodes, nodeInfo)
		}
	}

	c.JSON(http.StatusOK, nodes)
}
