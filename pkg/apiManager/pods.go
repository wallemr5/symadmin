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

//GetNodeProject
func (m *ApiManager) GetNodeProject(c *gin.Context) {
	// clusterName := c.Param("name")
	nodeIp := c.Param("ip")

	clusters := m.K8sMgr.GetAll()

	ctx := context.Background()
	pods := make([]*model.Project, 0, 4)

	listOptions := &client.ListOptions{}
	//listOptions.MatchingField("spec.nodeName",nodeIp)

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
			if pod.Status.HostIP == nodeIp {
				dm := pod.GetLabels()
				//if dm,ok := dm["lightningDomain0"];ok{
				if dm, ok := dm["app"]; ok {
					pods = append(pods, &model.Project{
						NodeIp:     pod.Status.HostIP,
						DomainName: dm,
						PodIp:      pod.Status.PodIP,
						PodCount:   len(podList.Items),
					})
				}
			}
		}

	}

	c.JSON(http.StatusOK, pods)
}
func (m *ApiManager) GetPod(c *gin.Context) {
	// clusterName := c.Param("name")

	appName := c.Param("appName")

	clusters := m.K8sMgr.GetAll()

	ctx := context.Background()
	pods := make([]*model.Pod, 0, 4)

	listOptions := &client.ListOptions{}
	listOptions.MatchingLabels(map[string]string{
		"app": appName,
	})
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
			pods = append(pods, &model.Pod{
				Name:         pod.Name,
				NodeIp:       pod.Status.HostIP,
				PodIp:        pod.Status.PodIP,
				ImageVersion: "",
			})
		}

	}

	c.JSON(http.StatusOK, pods)
}
