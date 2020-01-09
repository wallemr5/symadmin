package apiManager

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog"
)

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
	MsgType string `json:"type"`
	Input   string `json:"input"`
	Rows    uint16 `json:"rows"`
	Cols    uint16 `json:"cols"`
}

// GetTerminal ...
func (m *APIManager) GetTerminal(c *gin.Context) {
	clusterName := c.Param("name")
	namespace := c.DefaultQuery("namespace", "default")
	tty, _ := strconv.ParseBool(c.DefaultQuery("tty", "true"))
	isStdin, _ := strconv.ParseBool(c.DefaultQuery("stdin", "true"))
	isStdout, _ := strconv.ParseBool(c.DefaultQuery("stdout", "true"))
	isStderr, _ := strconv.ParseBool(c.DefaultQuery("stderr", "true"))
	once, _ := strconv.ParseBool(c.DefaultQuery("once", "false"))
	cmd := strings.Fields(c.Query("cmd"))

	podName, ok := c.GetQuery("pod")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": "", // TODO define error code
			"msg":  "can not get pod name.",
		})
		return
	}

	containerName, ok := c.GetQuery("container")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": "", // TODO define error code
			"msg":  "can not get container name.",
		})
		return
	}

	cluster, err := m.K8sMgr.Get(clusterName)
	if err != nil {
		klog.Errorf("get cluster error: %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code": "", // TODO define error code
			"msg":  err.Error(),
		})
		return
	}

	ws, err := InitWebsocket(c.Writer, c.Request)
	if err != nil {
		klog.Errorf("init websocket conn error: %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code": "", // TODO define error code
			"msg":  err.Error(),
		})
		return
	}

	err = startProcess(cluster, namespace, podName, containerName,
		cmd, isStdin, isStdout, isStderr, tty, once, ws)
	if err != nil {
		klog.Errorf("error in startProcess: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"code": "", // TODO define error code
			"msg":  err.Error(),
		})
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

	if !once {
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

// InitWebsocket ...
func InitWebsocket(resp http.ResponseWriter, req *http.Request) (ws *WsConnection, err error) {
	conn, err := wsUpgrader.Upgrade(resp, req, nil)
	if err != nil {
		return nil, err
	}
	ws = &WsConnection{
		conn:      conn,
		inChan:    make(chan *WsMessage, 1000),
		outChan:   make(chan *WsMessage, 1000),
		closeChan: make(chan byte),
		isClosed:  false,
	}

	go ws.ReadLoop()
	go ws.WriteLoop()

	return
}

// Next ...
func (handler *streamHandler) Next() (size *remotecommand.TerminalSize) {
	ret := <-handler.resizeEvent
	size = &ret
	return
}

// Read ...
func (handler *streamHandler) Read(p []byte) (size int, err error) {
	msg, err := handler.ws.Read()
	if err != nil {
		handler.ws.Close()
		return 0, err
	}

	xtermMsg := &xtermMessage{
		Input: string(msg.Data),
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
			klog.Errorf("readloop error: %v", err)
			break
		}
		ws.inChan <- &WsMessage{msgType, data}
	}
}

// WriteLoop ...
func (ws *WsConnection) WriteLoop() {
	for {
		select {
		case msg := <-ws.outChan:
			if err := ws.conn.WriteMessage(msg.MessageType, msg.Data); err != nil {
				klog.Errorf("error in write websocket message: %v", err)
				continue
			}
		case <-ws.closeChan:
			ws.Close()
			break
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

// Close ...
func (ws *WsConnection) Close() {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	if !ws.isClosed {
		ws.isClosed = true
		ws.closeChan <- 0
		close(ws.closeChan)
		ws.conn.Close()
	}
}
