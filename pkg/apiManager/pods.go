package apiManager

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//GetNodeProject ...
func (m *APIManager) GetNodeProject(c *gin.Context) {
	// clusterName := c.Param("name")
	nodeIP := c.Param("ip")

	clusters := m.K8sMgr.GetAll()

	ctx := context.Background()
	pods := &model.NodeProjects{}

	listOptions := &client.ListOptions{}
	//listOptions.MatchingField("spec.nodeName",nodeIP)

	for _, cluster := range clusters {
		if cluster.Status == k8smanager.ClusterOffline {
			continue
		}
		podList := &corev1.PodList{}
		err := cluster.Client.List(ctx, listOptions, podList)
		if err != nil {

			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Error(err, "failed to get pods")
			break
		}

		for i := range podList.Items {
			pod := &podList.Items[i]
			if pod.Status.HostIP == nodeIP {
				dm := pod.GetLabels()
				//if dm,ok := dm["lightningDomain0"];ok{
				if dm, ok := dm["app"]; ok {
					pods.Projects = append(pods.Projects, &model.Project{

						DomainName: dm,
						PodIP:      pod.Status.PodIP,
					})
				}
			}
		}

		pods.PodCount = len(pods.Projects)
		pods.NodeIP = nodeIP
	}
	c.JSON(http.StatusOK, pods)
}

// GetPod ...
func (m *APIManager) GetPod(c *gin.Context) {

	appName := c.Param("appName")

	clusters := m.K8sMgr.GetAll()

	ctx := context.Background()
	pods := &model.Pod{}

	listOptions := &client.ListOptions{}
	listOptions.MatchingLabels(map[string]string{
		"app": appName,
	})
	for _, cluster := range clusters {
		podList := &corev1.PodList{}
		err := cluster.Client.List(ctx, listOptions, podList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			klog.Error(err, "failed to get pods")
			break
		}

		for i := range podList.Items {
			pod := &podList.Items[i]
			//if pod.Name == appName{
			pods.ContainerStatus = append(pods.ContainerStatus, &model.ContainerStatus{
				Name:         pod.Status.ContainerStatuses[0].Name,
				Ready:        pod.Status.ContainerStatuses[0].Ready,
				RestartCount: pod.Status.ContainerStatuses[0].RestartCount,
				Image:        pod.Status.ContainerStatuses[0].Image,
				ContainerID:  pod.Status.ContainerStatuses[0].ContainerID,
			})
			pods.Name = pod.Name
			pods.NodeIP = pod.Status.HostIP
			pods.PodIP = pod.Status.PodIP
			pods.StartTime = pod.Status.StartTime.String()
			//}
		}
	}
	c.JSON(http.StatusOK, pods)
}
