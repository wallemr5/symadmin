package v1

import (
	"context"
	"errors"
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
	clusterName := c.Param("name")
	podName := c.Param("podName")
	namespace := c.Param("namespace")
	container := c.DefaultQuery("container", "")
	tailLines, _ := strconv.ParseInt(c.DefaultQuery("tail", "1000"), 10, 64)
	// limitBytes, _ := strconv.ParseInt(c.DefaultQuery("limitBytes", "2048"), 10, 64)
	follow, _ := strconv.ParseBool(c.DefaultQuery("follow", "false"))
	previous, _ := strconv.ParseBool(c.DefaultQuery("previous", "false"))
	timestamps, _ := strconv.ParseBool(c.DefaultQuery("timestamps", "false"))
	// sinceSecond, _ := strconv.ParseInt(c.DefaultQuery("sinceSecond", "1"), 10, 64)

	logOptions := &corev1.PodLogOptions{
		Follow:     follow,
		Previous:   previous,
		Timestamps: timestamps,
		TailLines:  &tailLines,
		// LimitBytes: &limitBytes,
	}

	// sinceTime := metav1.Time{}
	// sinceTimeStr, ok := c.GetQuery("sinceTime")
	// if ok {
	// 	parseTime, err := time.Parse("", sinceTimeStr)
	// 	sinceTime.Time = parseTime
	// 	logOptions.SinceTime = &sinceTime
	// 	if err != nil {
	// 		AbortHTTPError(c, ParseTimeStampError, "", err)
	// 		return
	// 	}
	// }
	// else {
	// 	logOptions.SinceSeconds = &sinceSecond
	// }

	cluster, err := m.ClustersMgr.Get(clusterName)
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
		AbortHTTPError(c, GetPodLogsError, "", err)
		return
	}
	defer req.Close()

	result, err := ioutil.ReadAll(req)
	if err != nil {
		klog.Errorf("get pod log error: %v", err)
		AbortHTTPError(c, GetPodLogsError, "", err)
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
		"resultMap": gin.H{
			"log": processTextToHTML(string(result)),
		}})
}

// HandleFileLogs get log files in a pod
func (m *Manager) HandleFileLogs(c *gin.Context) {
	clusterName := c.Param("name")
	namespace := c.Param("namespace")
	podName := c.Param("podName")
	tailLines := c.DefaultQuery("tailLines", "1000")

	containerName, ok := c.GetQuery("container")
	if !ok {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("can not get container"))
		return
	}
	filepath, ok := c.GetQuery("filepath")
	if !ok {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("can not get filename"))
		return
	}

	cluster, err := m.ClustersMgr.Get(clusterName)
	if err != nil {
		klog.Errorf("get cluster error: %+v", err)
		AbortHTTPError(c, GetClusterError, "", err)
		return
	}

	cmd := "tail -n " + tailLines + " " + filepath
	result, err := RunCmdOnceInContainer(cluster, namespace, podName, containerName, cmd, false)
	if err != nil {
		klog.Errorf("run cmd once in container error: %v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
			"resultMap": gin.H{
				"path":   filepath,
				"applog": "",
			},
		})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
		"resultMap": gin.H{
			"path":   filepath,
			"applog": processTextToHTML(string(result)),
		},
	})
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
