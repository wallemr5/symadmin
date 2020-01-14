package apiManager

import (
	"context"
	"io/ioutil"
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
	podName := c.Param("podName")
	namespace := c.DefaultQuery("namespace", "default")
	container := c.DefaultQuery("container", "")
	tailLines, _ := strconv.ParseInt(c.DefaultQuery("tail", "10"), 10, 64)
	limitBytes, _ := strconv.ParseInt(c.DefaultQuery("limitBytes", "2048"), 10, 64)
	follow, _ := strconv.ParseBool(c.DefaultQuery("follow", "true"))
	previous, _ := strconv.ParseBool(c.DefaultQuery("previous", "false"))
	timestamps, _ := strconv.ParseBool(c.DefaultQuery("timestamps", "true"))
	// sinceSecond, _ := strconv.ParseInt(c.DefaultQuery("sinceSecond", "1"), 10, 64)

	logOptions := &corev1.PodLogOptions{
		Follow:     follow,
		Previous:   previous,
		Timestamps: timestamps,
		TailLines:  &tailLines,
		LimitBytes: &limitBytes,
	}

	sinceTime := metav1.Time{}
	sinceTimeStr, ok := c.GetQuery("sinceTime")
	if ok {
		parseTime, err := time.Parse("", sinceTimeStr)
		sinceTime.Time = parseTime
		logOptions.SinceTime = &sinceTime
		if err != nil {
			AbortHTTPError(c, ParseTimeStampError, "", err)
			return
		}
	}
	// else {
	// 	logOptions.SinceSeconds = &sinceSecond
	// }

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
		logOptions.Container = pod.Spec.Containers[0].Name
	}

	req, err := cluster.KubeCli.CoreV1().RESTClient().Get().
		Namespace(namespace).
		Name(podName).
		Resource("pods").
		SubResource("log").
		VersionedParams(logOptions, scheme.ParameterCodec).
		Stream()
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

	c.IndentedJSON(http.StatusOK, gin.H{"logs": string(result)})
}

// HandleFileLogs get log files in a pod
func (m *APIManager) HandleFileLogs(c *gin.Context) {
}
