package v2

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/klog"
)

// File ...
type File struct {
	Name        string `json:"name"`
	FullPath    string `json:"fullPath"`
	IsDirectory bool   `json:"isDirectory"`
}

// TailFile tail files in a pod
func (m *Manager) TailFile(c *gin.Context) {
	clusterName := c.Param("clusterCode")
	namespace := c.Param("namespace")
	podName := c.Param("podName")
	tailLines := c.DefaultQuery("tail", "1000")

	containerName, ok := c.GetQuery("container")
	if !ok || containerName == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   "container can not be none",
			"resultMap": nil,
		})
		return
	}
	filepath, ok := c.GetQuery("filepath")
	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   "filename can not be none",
			"resultMap": nil,
		})
		return
	}

	cluster, err := m.K8sMgr.Get(clusterName)
	if err != nil {
		klog.Errorf("get cluster error: %+v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}

	cmd := []string{
		"sh",
		"-c",
		"tail -n" + tailLines + " " + filepath,
	}
	result, err := RunCmdOnceInContainer(cluster, namespace, podName, containerName, cmd, false)
	if err != nil {
		klog.Errorf("run cmd once in container error: %v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
			"resultMap": gin.H{
				"path": filepath,
				"data": "",
			},
		})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
		"resultMap": gin.H{
			"path": filepath,
			"data": string(result),
		},
	})
}

// ListFiles get the log file of the specified directory
// use cmd: ls -p
// isDirectory is true when fileName has suffix '/'
func (m *Manager) ListFiles(c *gin.Context) {
	clusterCode := c.Param("clusterCode")
	namespace := c.Param("namespace")
	podName := c.Param("podName")
	path := c.DefaultQuery("path", "/web/logs/app/")
	if path == "" {
		path = "/web/logs/app/"
	}

	containerName, ok := c.GetQuery("container")
	if !ok || containerName == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   "container can not be none",
			"resultMap": nil,
		})
		return
	}
	cluster, err := m.K8sMgr.Get(clusterCode)
	if err != nil {
		klog.Errorf("get cluster error: %+v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"message":   err.Error(),
			"resultMap": nil,
		})
		return
	}

	cmd := []string{
		"sh",
		"-c",
		"ls -p " + path,
	}
	cmdResult, err := RunCmdOnceInContainer(
		cluster, namespace, podName, containerName, cmd, false)
	if err != nil {
		klog.Errorf("run cmd once in container error: %v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
			"resultMap": gin.H{
				"data": nil,
				"path": path,
			},
		})
		return
	}

	files := strings.Split(string(cmdResult), "\n")
	result := []*File{}
	for _, fileName := range files {
		if fileName == "" {
			continue
		}

		f := &File{Name: fileName}
		if strings.HasSuffix(fileName, "/") {
			f.IsDirectory = true
			f.Name = strings.TrimSuffix(fileName, "/")
		}
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
		f.FullPath = path + f.Name
		result = append(result, f)
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"resultMap": gin.H{
			"data": result,
			"path": path,
		},
	})
}
