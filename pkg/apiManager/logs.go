package apiManager

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
)

// HandleLogs get the pod container stdout
func (m *APIManager) HandleLogs(c *gin.Context) {
	clusterName := c.Param("name")
	podName := c.Param("appName")
	namespace := c.DefaultQuery("namespace", "default")
	container := c.DefaultQuery("container", "")
	tailLines, _ := strconv.ParseInt(c.DefaultQuery("tail", "100"), 10, 64)

	cluster, err := m.K8sMgr.Get(clusterName)
	if err != nil {
		AbortHTTPError(c, GetClusterError, "", err)
		return
	}

	ctx := context.Background()
	pod := &corev1.Pod{}
	err = cluster.Client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      podName,
	}, pod)
	if err != nil {
		klog.Errorf("get pod error: %v", err)
		AbortHTTPError(c, GetPodError, "", err)
		return
	}

	if len(container) == 0 {
		container = pod.Spec.Containers[0].Name
	}

	logOptions := &corev1.PodLogOptions{
		Container:    container,
		Follow:       false,
		Previous:     false,
		SinceSeconds: nil,
		SinceTime: &metav1.Time{
			Time: time.Time{},
		},
		Timestamps: false,
		TailLines:  &tailLines,
		LimitBytes: nil,
	}
	logs, err := cluster.KubeCli.CoreV1().RESTClient().Get().
		Namespace(namespace).
		Name(podName).
		Resource("pods").
		SubResource("log").
		VersionedParams(logOptions, scheme.ParameterCodec).
		Stream()
	if err != nil {
		klog.Errorf("get pod error: %v", err)
		AbortHTTPError(c, GetPodLogsError, "", err)
		return
	}

	c.JSON(http.StatusOK, logs)
}

// HandleFileLogs get log files in a pod
func (m *APIManager) HandleFileLogs(c *gin.Context) {
	// clusterName := c.Param("name")
	// podName := c.Param("appName")
	// namespace := c.DefaultQuery("namespace", "default")
	// container := c.DefaultQuery("container", "")
	// tailLines, _ := strconv.ParseInt(c.DefaultQuery("tail", "100"), 10, 64)

	// cluster, err := m.K8sMgr.Get(clusterName)
	// if err != nil {
	// 	AbortHTTPError(c, GetClusterError, "", err)
	// 	return
	// }
}
