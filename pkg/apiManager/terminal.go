package apiManager

import (
	"net/http"
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
func (m *ApiManager) GetTerminal(c *gin.Context) {
	clusterName := c.Param("name")
	namespace := c.DefaultQuery("namespace", "default")
	podName, ok := c.GetQuery("pod")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "can not get pod name."})
	}

	containerName, ok := c.GetQuery("container")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "can not get container name."})
	}

	cluster, err := m.K8sMgr.Get(clusterName)
	if err != nil {
		klog.Errorf("get cluster error: %+v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "can not get cluster."})
	}

	ws, err := InitWebsocket(c.Writer, c.Request)
	if err != nil {
		klog.Errorf("init websocket conn error: %+v", err)
	}
	defer ws.Close()

	// TODO(haidong): support more shell,bash/powershell
	err = startProcess(ws, cluster, podName, namespace, containerName, []string{"/bin/sh"})
	if err != nil {
		klog.Errorf("error in startProcess: %v", err)
	}
}

func startProcess(ws *WsConnection, cluster *k8smanager.Cluster, podName, namespace, container string, cmd []string) error {
	req := cluster.KubeCli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	scheme := runtime.NewScheme()
	if err := core_v1.AddToScheme(scheme); err != nil {
		klog.Errorf("error adding to scheme: %v", err)
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&core_v1.PodExecOptions{
		Container: container,
		Command:   cmd,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(cluster.RestConfig, "POST", req.URL())
	if err != nil {
		ws.Close()
		klog.Errorf("error while creating Executor: %+v", err)
		return err

	}

	handler := &streamHandler{
		ws:          ws,
		resizeEvent: make(chan remotecommand.TerminalSize),
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:             handler,
		Stdout:            handler,
		Stderr:            handler,
		TerminalSizeQueue: handler,
		Tty:               true,
	})
	if err != nil {
		klog.Errorf("error in Stream: %+v", err)
		return err

	}

	return err
}

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

func (handler *streamHandler) Next() (size *remotecommand.TerminalSize) {
	ret := <-handler.resizeEvent
	size = &ret
	return
}

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

func (ws *WsConnection) ReadLoop() {
	for {
		msgType, data, err := ws.conn.ReadMessage()
		if err != nil {
			klog.Errorf("error in read websocket message: %v", err)
			continue
		}
		ws.inChan <- &WsMessage{msgType, data}
	}
}

func (ws *WsConnection) WriteLoop() {
	for {
		select {
		case msg := <-ws.outChan:
			if err := ws.conn.WriteMessage(msg.MessageType, msg.Data); err != nil {
				klog.Errorf("error in write websocket message: %v", err)
			}
		case <-ws.closeChan:
			ws.Close()
		}
	}

}

func (ws *WsConnection) Write(messageType int, data []byte) (err error) {
	select {
	case ws.outChan <- &WsMessage{messageType, data}:
	case <-ws.closeChan:
		klog.Info("websocket closed in write")
		break
	}
	return nil
}

func (ws *WsConnection) Read() (msg *WsMessage, err error) {
	select {
	case msg := <-ws.inChan:
		return msg, nil
	case <-ws.closeChan:
		klog.Info("websocket closed in read")
	}
	return nil, nil
}

func (ws *WsConnection) Close() {
	ws.conn.Close()
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	if !ws.isClosed {
		ws.isClosed = true
		close(ws.closeChan)
	}
}
