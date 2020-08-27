package v2

import (
	"context"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
)

// HandleLogs get the pod container stdout
func (m *Manager) HandleLogs(c *gin.Context) {
	clusterCode := c.Param("clusterCode")
	podName := c.Param("podName")
	namespace := c.Param("namespace")
	container := c.DefaultQuery("container", "")

	tailStr := c.DefaultQuery("tail", "1000")
	if tailStr == "" {
		tailStr = "1000"
	}
	tailLines, err := strconv.ParseInt(tailStr, 10, 64)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}

	follow, _ := strconv.ParseBool(c.DefaultQuery("follow", "false"))
	previous, _ := strconv.ParseBool(c.DefaultQuery("previous", "false"))
	timestamps, _ := strconv.ParseBool(c.DefaultQuery("timestamps", "false"))

	logOptions := &corev1.PodLogOptions{
		Follow:     follow,
		Previous:   previous,
		Timestamps: timestamps,
		TailLines:  &tailLines,
	}

	cluster, err := m.K8sMgr.Get(clusterCode)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
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
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}

	if len(container) == 0 {
		logOptions.Container = pod.Spec.Containers[0].Name
	} else {
		logOptions.Container = container
	}

	req, err := cluster.KubeCli.CoreV1().RESTClient().Get().
		Namespace(namespace).
		Name(podName).
		Resource("pods").
		SubResource("log").
		VersionedParams(logOptions, scheme.ParameterCodec).
		Stream(context.TODO())
	if err != nil {
		klog.Errorf("get pod log error: %v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}
	defer req.Close()

	result, err := ioutil.ReadAll(req)
	if err != nil {
		klog.Errorf("get pod log error: %v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
		"resultMap": gin.H{
			"log": processTextToHTML(string(result)),
		}})
}

func processTextToHTML(text string) string {
	if text == "" {
		return "暂无日志"
	}
	split := strings.Split(strings.Replace(text, "\r\n", "\n", -1), "\n")

	var result []string
	for _, s := range split {
		if strings.Contains(s, "WARNING") || strings.Contains(s, "WARN") {
			s = "<span class=\"text-warning\">" + s + "</span>"
		} else if strings.Contains(s, "ERROR") || strings.Contains(s, "FAILURE") || strings.Contains(s, "Exception") {
			s = "<span class=\"text-danger\">" + s + "</span>"
		}
		result = append(result, s)
	}
	return strings.Join(result, "<br/>")
}
