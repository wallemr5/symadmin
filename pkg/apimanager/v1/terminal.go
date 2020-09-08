package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var wsMap sync.Map

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WsMessage ...
type WsMessage struct {
	MessageType int
	Data        []byte
}

// WsConnection ...
type WsConnection struct {
	id        uint32
	conn      *websocket.Conn
	inChan    chan *WsMessage
	outChan   chan *WsMessage
	mutex     sync.Mutex
	isClosed  bool
	closeChan chan byte
}

type streamHandler struct {
	ws          *WsConnection
	resizeEvent chan remotecommand.TerminalSize
}

type xtermMessage struct {
	MsgType string `json:"type,omitempty"`
	Input   string `json:"input,omitempty"`
	Rows    uint16 `json:"rows,omitempty"`
	Cols    uint16 `json:"cols,omitempty"`
}

// GetTerminal ...
func (m *Manager) GetTerminal(c *gin.Context) {
	clusterName := c.Param("name")
	namespace := c.DefaultQuery("namespace", "default")
	tty, _ := strconv.ParseBool(c.DefaultQuery("tty", "true"))
	isStdin, _ := strconv.ParseBool(c.DefaultQuery("stdin", "true"))
	isStdout, _ := strconv.ParseBool(c.DefaultQuery("stdout", "true"))
	isStderr, _ := strconv.ParseBool(c.DefaultQuery("stderr", "true"))
	once, _ := strconv.ParseBool(c.DefaultQuery("once", "false"))
	cmd := c.QueryArray("cmd")

	podName, ok := c.GetQuery("pod")
	if !ok {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("can not get pod"))
		return
	}

	containerName, ok := c.GetQuery("container")
	if !ok {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("can not get container"))
		return
	}

	cluster, err := m.ClustersMgr.Get(clusterName)
	if err != nil {
		klog.Errorf("get cluster error: %+v", err)
		AbortHTTPError(c, GetClusterError, "", err)
		return
	}

	rand.Seed(time.Now().UnixNano())
	ws, err := InitWebsocket(c.Writer, c.Request, rand.Uint32())
	if err != nil {
		klog.Errorf("init websocket conn error: %+v", err)
		AbortHTTPError(c, WebsocketError, "", err)
		return
	}
	defer ws.Close()

	err = startProcess(cluster, namespace, podName, containerName,
		cmd, isStdin, isStdout, isStderr, tty, once, ws)
	if err != nil {
		ws.Write(websocket.BinaryMessage, []byte(err.Error()+". "))
		ws.closeChan <- 0
	}
}

// ExecOnceWithHTTP ...
func (m *Manager) ExecOnceWithHTTP(c *gin.Context) {
	clusterName := c.Param("name")
	namespace := c.DefaultQuery("namespace", "default")
	tty, _ := strconv.ParseBool(c.DefaultQuery("tty", "false"))
	podName, ok := c.GetQuery("pod")
	if !ok {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("can not get pod name"))
		return
	}

	cmd, ok := c.GetQuery("cmd")
	if !ok {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("no command to exec"))
		return
	}

	containerName, ok := c.GetQuery("container")
	if !ok {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("can not get container"))
		return
	}

	cluster, err := m.ClustersMgr.Get(clusterName)
	if err != nil {
		klog.Errorf("get cluster error: %+v", err)
		AbortHTTPError(c, GetClusterError, "", err)
		return
	}

	result, err := RunCmdOnceInContainer(cluster, namespace, podName, containerName, cmd, tty)
	if err != nil {
		klog.Errorf("run cmd once in container error: %v", err)
		AbortHTTPError(c, ExecCmdError, "", err)
		return
	}

	c.IndentedJSON(http.StatusOK, result)
}

// GetFiles get the log file of the specified directory
func (m *Manager) GetFiles(c *gin.Context) {
	clusterCode := c.Query("clusterCode")
	namespace := c.Query("namespace")
	podName := c.Query("podName")
	projectCode := c.Query("projectCode")
	appCode := c.Query("appCode")
	appName := c.Query("appName")

	containerName, ok := c.GetQuery("container")
	if !ok {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("can not get container"))
		return
	}
	cluster, err := m.ClustersMgr.Get(clusterCode)
	if err != nil {
		klog.Errorf("get cluster error: %+v", err)
		AbortHTTPError(c, GetClusterError, "", err)
		return
	}

	// Get podIP and containerID
	ctx := context.Background()
	pod := &core_v1.Pod{}
	err = cluster.Client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      podName,
	}, pod)
	if err != nil {
		klog.Errorf("get pod error: %v", err)
		AbortHTTPError(c, GetPodError, "", err)
		return
	}

	podIP := pod.Status.PodIP
	var containerID string
	for _, c := range pod.Status.ContainerStatuses {
		if c.Name == containerName {
			containerID = c.ContainerID
		}
	}

	// New logging rules: /web/logs/app/$projectCode/$appCode/$ip:$port/*.log
	path := fmt.Sprintf("/web/logs/app/%s/%s/", projectCode, appCode)
	cmd := "ls " + path
	result, err := RunCmdOnceInContainer(
		cluster, namespace, podName, containerName, cmd, false)
	if err != nil {
		klog.Errorf("run cmd once in container error: %v", err)
		c.IndentedJSON(http.StatusOK, gin.H{
			"success": true,
			"message": nil,
			"resultMap": gin.H{
				"result":    []string{},
				"errorText": []string{err.Error()},
				"path":      path,
				"success":   false,
			},
		})
		return
	}
	files := strings.Split(string(result), "\n")
	var logDirectory string
	for _, fileName := range files {
		if strings.HasPrefix(fileName, podIP) {
			logDirectory = fileName
			break
		}
	}

	if len(logDirectory) > 0 {
		cmd += logDirectory
		result, err = RunCmdOnceInContainer(
			cluster, namespace, podName, containerName, cmd, false)
		if err != nil {
			klog.Errorf("run cmd once in container error: %v", err)
			c.IndentedJSON(http.StatusOK, gin.H{
				"success": true,
				"message": nil,
				"resultMap": gin.H{
					"result":    []string{},
					"errorText": []string{err.Error()},
					"path":      path + logDirectory,
					"success":   false,
				},
			})
			return
		}
		files := strings.Split(string(result), "\n")
		if len(files) > 1 {
			c.IndentedJSON(http.StatusOK, gin.H{
				"success": true,
				"message": nil,
				"resultMap": gin.H{
					"result":    files[:len(files)-1],
					"errorText": nil,
					"path":      path + logDirectory,
					"success":   true,
				},
			})
			return
		}
	}

	// Old logging rules: /web/logs/app/logback/$appName/$podIP_$containerID/
	path = fmt.Sprintf("/web/logs/app/logback/%s/", appName)
	cmd = "ls " + path
	result, err = RunCmdOnceInContainer(
		cluster, namespace, podName, containerName, cmd, false)
	if err != nil {
		klog.Errorf("run cmd once in container error: %v", err)
		c.IndentedJSON(http.StatusOK, gin.H{
			"success": true,
			"message": nil,
			"resultMap": gin.H{
				"result":    []string{},
				"errorText": []string{err.Error()},
				"path":      path,
				"success":   false,
			},
		})
		return
	}
	files = strings.Split(string(result), "\n")
	for _, fileName := range files {
		if strings.HasPrefix(fileName, podIP+"_"+containerID[9:12]) || strings.HasPrefix(fileName, podIP+"_docker-"+containerID[9:12]) {
			logDirectory = fileName
			break
		}
	}
	if len(logDirectory) > 0 {
		cmd += logDirectory
		result, err = RunCmdOnceInContainer(
			cluster, namespace, podName, containerName, cmd, false)
		if err != nil {
			klog.Errorf("run cmd once in container error: %v", err)
			c.IndentedJSON(http.StatusOK, gin.H{
				"success": true,
				"message": nil,
				"resultMap": gin.H{
					"result":    []string{},
					"errorText": []string{err.Error()},
					"path":      path + logDirectory,
					"success":   false,
				},
			})
			return
		}
		files = strings.Split(string(result), "\n")
		if len(files) > 1 {
			c.IndentedJSON(http.StatusOK, gin.H{
				"success": true,
				"message": nil,
				"resultMap": gin.H{
					"result":    files[:len(files)-1],
					"errorText": nil,
					"path":      path + logDirectory,
					"success":   true,
				},
			})
			return
		}
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": "no log files found.",
		"resultMap": gin.H{
			"result":    []string{},
			"errorText": []string{"no log files found."},
			"path":      path,
			"success":   false,
		},
	})
}

// GetOfflineLogTerminal ...
func (m *Manager) GetOfflineLogTerminal(c *gin.Context) {
	clusterName := c.Param("name")
	namespace := "sym-admin"

	tty, _ := strconv.ParseBool(c.DefaultQuery("tty", "true"))
	isStdin, _ := strconv.ParseBool(c.DefaultQuery("stdin", "true"))
	isStdout, _ := strconv.ParseBool(c.DefaultQuery("stdout", "true"))
	isStderr, _ := strconv.ParseBool(c.DefaultQuery("stderr", "true"))
	once, _ := strconv.ParseBool(c.DefaultQuery("once", "false"))

	var podName, containerName string

	lb := labels.Set{
		"app": "offline-pod-log",
	}

	hostIP, ok := c.GetQuery("hostIP")
	if !ok {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("can not get hostIP"))
		return
	}

	offlinePodIP := c.DefaultQuery("offlinePodIP", "")
	containerID := c.DefaultQuery("containerID", "")
	projectCode, ok := c.GetQuery("projectCode")
	if !ok {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("can not get projectCode"))
		return
	}

	appCode, ok := c.GetQuery("appCode")
	if !ok {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("can not get appcCode"))
		return
	}

	path := fmt.Sprintf("/web/logs/app/%s/%s", projectCode, appCode)

	cluster, err := m.ClustersMgr.Get(clusterName)
	if err != nil {
		klog.Errorf("get cluster error: %+v", err)
		AbortHTTPError(c, GetClusterError, "", err)
		return
	}

	listOptions := &client.ListOptions{Namespace: namespace, LabelSelector: lb.AsSelector()}

	podList, err := m.Cluster.GetPods(listOptions, clusterName)
	if err != nil {
		AbortHTTPError(c, ParamInvalidError, "", errors.New("can not get offlineDepoy"))
		return
	}
	for _, pod := range podList {
		if hostIP == pod.Status.HostIP {
			podName = pod.Name
			containerName = pod.Status.ContainerStatuses[0].Name
			break
		}
	}
	if podName == "" || containerName == "" {
		AbortHTTPError(c, GetPodNotGroup, "", errors.New("can not get offlinepod"))
		return
	}
	rand.Seed(time.Now().UnixNano())
	ws, err := InitWebsocket(c.Writer, c.Request, rand.Uint32())
	if err != nil {
		klog.Errorf("init websocket conn error: %+v", err)
		AbortHTTPError(c, WebsocketError, "", err)
		return
	}
	defer ws.Close()
	//oldpath /web/logs/app/logback/$appName/$podIP_$containerID/
	//newpath /web/logs/app/$projectName/$appName/$podIP:$Port
	if offlinePodIP != "" {
		if containerID == "" {
			path = fmt.Sprintf("/web/logs/app/%s/%s/%s*", projectCode, appCode, offlinePodIP)
		} else {
			path = fmt.Sprintf("/web/logs/app/logback/%s/%s_%s", appCode, offlinePodIP, containerID)
		}
	}

	logshell := fmt.Sprintf("cd %s ;pwd &&/bin/bash", path)
	cmd := []string{
		"sh",
		"-c",
		logshell,
	}

	err = startProcess(cluster, namespace, podName, containerName,
		cmd, isStdin, isStdout, isStderr, tty, once, ws)
	if err != nil {
		ws.Write(websocket.BinaryMessage, []byte(err.Error()+". "))
		ws.closeChan <- 0
	}
}

func startProcess(cluster *k8smanager.Cluster, namespace, podName, container string,
	cmd []string, isStdin, isStdout, isStderr, tty, once bool, ws *WsConnection) error {
	req := cluster.KubeCli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	scheme := runtime.NewScheme()
	if err := core_v1.AddToScheme(scheme); err != nil {
		klog.Errorf("error adding to scheme: %v", err)
		return err
	}

	if !once && len(cmd) == 0 {
		cmd = []string{"/bin/sh"}
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&core_v1.PodExecOptions{
		Command:   cmd,
		Container: container,
		Stdin:     isStdin,
		Stdout:    isStdout,
		Stderr:    isStderr,
		TTY:       tty,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(cluster.RestConfig, "POST", req.URL())
	if err != nil {
		klog.Errorf("error while creating Executor: %+v", err)
		return err

	}

	if once {
		var stdout, stderr bytes.Buffer
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:  nil,
			Stdout: &stdout,
			Stderr: &stderr,
			Tty:    tty,
		})
		if stderr.Len() != 0 {
			klog.Errorf("exec steam error: %v", stderr)
			ws.Write(websocket.TextMessage, stderr.Bytes())
		} else {
			ws.Write(websocket.TextMessage, stdout.Bytes())
		}
		ws.Close()
	} else {
		handler := &streamHandler{
			ws:          ws,
			resizeEvent: make(chan remotecommand.TerminalSize),
		}
		err = exec.Stream(remotecommand.StreamOptions{
			Stdin:             handler,
			Stdout:            handler,
			Stderr:            handler,
			TerminalSizeQueue: handler,
			Tty:               tty,
		})
	}
	if err != nil {
		klog.Errorf("error in Stream: %+v", err)
		return err
	}

	return err
}

// RunCmdOnceInContainer ...
func RunCmdOnceInContainer(cluster *k8smanager.Cluster, namespace, pod, container, cmd string, tty bool) ([]byte, error) {
	req := cluster.KubeCli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec")

	scheme := runtime.NewScheme()
	if err := core_v1.AddToScheme(scheme); err != nil {
		klog.Errorf("error adding to scheme: %v", err)
		return nil, err
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	cmd = strings.ReplaceAll(cmd, "'", "")
	klog.Infof("exec cmd: %s", cmd)
	req.VersionedParams(&core_v1.PodExecOptions{
		Command:   strings.Fields(cmd),
		Container: container,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       tty,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(cluster.RestConfig, "POST", req.URL())
	if err != nil {
		klog.Errorf("error while creating Executor: %+v", err)
		return nil, err
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    tty,
	})
	if stderr.Len() > 0 {
		return nil, errors.New(stderr.String())
	}

	if err != nil {
		klog.Errorf("get exec streaming error: %v", err)
		return nil, err
	}

	return stdout.Bytes(), nil
}

// InitWebsocket ...
func InitWebsocket(resp http.ResponseWriter, req *http.Request, id uint32) (ws *WsConnection, err error) {
	conn, err := wsUpgrader.Upgrade(resp, req, nil)
	if err != nil {
		return nil, err
	}
	ws = &WsConnection{
		id:        id,
		conn:      conn,
		inChan:    make(chan *WsMessage, 1000),
		outChan:   make(chan *WsMessage, 1000),
		closeChan: make(chan byte),
		isClosed:  false,
	}
	wsMap.Store(id, &ws)
	klog.Infof("The total number of current websocket connections is: %d", getWsMapCount())

	go ws.ReadLoop()
	go ws.WriteLoop()

	return ws, nil
}

// Next ...
func (handler *streamHandler) Next() *remotecommand.TerminalSize {
	ret := <-handler.resizeEvent
	return &ret
}

// Read ...
func (handler *streamHandler) Read(p []byte) (size int, err error) {
	if handler.ws.isClosed {
		return 0, io.EOF
	}
	msg, err := handler.ws.Read()
	if err != nil {
		handler.ws.Close()
		return 0, err
	}

	if msg == nil {
		handler.ws.Close()
		return 0, err
	}

	xtermMsg := &xtermMessage{}
	if strings.HasPrefix(string(msg.Data), "___") {
		err := json.Unmarshal([]byte(string(msg.Data[3:])), xtermMsg)
		if err != nil {
			klog.Errorf("unmarshal ws msg to json error: %v", err)
			return 0, err
		}
	} else {
		xtermMsg.Input = string(msg.Data)
	}
	handler.resizeEvent <- remotecommand.TerminalSize{Width: xtermMsg.Cols, Height: xtermMsg.Rows}
	size = len(xtermMsg.Input)
	copy(p, xtermMsg.Input)
	return size, nil

}

// Write ...
func (handler *streamHandler) Write(p []byte) (size int, err error) {
	copyData := make([]byte, len(p))
	copy(copyData, p)
	size = len(p)
	err = handler.ws.Write(websocket.TextMessage, copyData)
	if err != nil {
		handler.ws.Close()
		return 0, err
	}
	return size, nil
}

// ReadLoop ...
func (ws *WsConnection) ReadLoop() {
	for {
		msgType, data, err := ws.conn.ReadMessage()
		if err != nil {
			if _, ok := err.(*websocket.CloseError); ok {
				klog.Info("websocket closed")
				ws.Close()
				break
			}
			klog.Errorf("readloop error: %v", err)
			break
		}
		ws.inChan <- &WsMessage{msgType, data}
	}
}

// WriteLoop ...
func (ws *WsConnection) WriteLoop() {
Loop:
	for {
		select {
		case msg := <-ws.outChan:
			// remove invalid UTF-8 characters
			msg.Data = []byte(strings.ToValidUTF8(string(msg.Data), ""))
			if err := ws.conn.WriteMessage(msg.MessageType, msg.Data); err != nil {
				klog.Errorf("error in write websocket message: %v", err)
				break Loop
			}
		case <-ws.closeChan:
			ws.Close()
			break Loop
		}
	}

}

// Write ...
func (ws *WsConnection) Write(messageType int, data []byte) (err error) {
	select {
	case ws.outChan <- &WsMessage{messageType, data}:
	case <-ws.closeChan:
		ws.Close()
	}
	return nil
}

// Read ...
func (ws *WsConnection) Read() (msg *WsMessage, err error) {
	select {
	case msg := <-ws.inChan:
		return msg, nil
	case <-ws.closeChan:
		ws.Close()
	}
	return nil, nil
}

func getWsMapCount() int {
	length := 0
	wsMap.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	return length
}

// Close ...
func (ws *WsConnection) Close() {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	wsMap.Delete(ws.id)
	if !ws.isClosed {
		ws.isClosed = true
		close(ws.closeChan)
		ws.conn.Close()
		klog.Infof("The total number of current websocket connections is: %d", getWsMapCount())
	}
}
