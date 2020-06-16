package model

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterStatus ...
type ClusterStatus struct {
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

// ContainerStatus ...
type ContainerStatus struct {
	Name         string                           `json:"name,omitempty"`
	Ready        bool                             `json:"ready,omitempty"`
	RestartCount int32                            `json:"restartCount,omitempty"`
	Image        string                           `json:"image,omitempty"`
	ContainerID  string                           `json:"containerId,omitempty"`
	LastState    *corev1.ContainerStateTerminated `json:"lastState,omitempty"`
}

// Pod ...
type Pod struct {
	Name         string             `json:"name,omitempty"`
	Namespace    string             `json:"namespace,omitempty"`
	ClusterCode  string             `json:"clusterCode,omitempty"`
	Annotations  map[string]string  `json:"annotations,omitempty"`
	HostIP       string             `json:"hostIP,omitempty"`
	Phase        corev1.PodPhase    `json:"phase,omitempty"`
	Group        string             `json:"group,omitempty"`
	RestartCount int32              `json:"restartCount"`
	PodIP        string             `json:"podIP,omitempty"`
	ImageVersion string             `json:"imageVersion,omitempty"`
	CommitID     string             `json:"commitId,omitempty"`
	StartTime    string             `json:"startTime,omitempty"`
	Labels       map[string]string  `json:"labels,omitempty"`
	Containers   []*ContainerStatus `json:"containers,omitempty"`
	Endpoints    bool               `json:"endpoints,omitempty"`
	HasLastState bool               `json:"hasLastState,omitempty"`
}

// ErrorResponse describes responses when an error occurred
type ErrorResponse struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Success bool   `json:"success"`
}

// Project ...
type Project struct {
	AppName    string   `json:"appName,omitempty"`
	DomainName string   `json:"domainName,omitempty"`
	PodCount   int      `json:"podCount,omitempty"`
	Instances  []string `json:"instances,omitempty"`
}

// NodeProjects ...
type NodeProjects struct {
	ClusterName string     `json:"clusterCode,omitempty"`
	NodeName    string     `json:"nodeName,omitempty"`
	NodeIP      string     `json:"nodeIp,omitempty"`
	PodCount    int        `json:"podCount,omitempty"`
	Projects    []*Project `json:"projects,omitempty"`
}

// Endpoints ...
type Endpoint struct {
	ClusterCode       string            `json:"clusterCode,omitempty"`
	Name              string            `json:"name,omitempty"`
	Namespace         string            `json:"namespace,omitempty"`
	CreationTimestamp string            `json:"creationTimes,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Release           string            `json:"release,omitempty"`
	Subsets           []string          `json:"subsets,omitempty"`
	TargetRefName     string            `json:"targetRefName,omitempty"`
}

// NodeInfo ...
type NodeInfo struct {
	Name          string `json:"name,omitempty"`
	ClusterName   string `json:"clusterCode,omitempty"`
	HostIP        string `json:"hostIP,omitempty"`
	KernelVersion string `json:"kernelVersion,omitempty"`
	Architecture  string `json:"architecture,omitempty"`
	MemorySize    int64  `json:"memorySize,omitempty"`
	Status        string `json:"status"`
	CPU           int64  `json:"cpu,omitempty"`
	JoinDate      string `json:"joinDate,omitempty"`
	System        string `json:"system,omitempty"`
	DockerVersion string `json:"dockerVersion,omitempty"`
}

// ServiceInfo ...
type ServiceInfo struct {
	Name        string               `json:"name,omitempty"`
	ClusterName string               `json:"clusterCode,omitempty"`
	NameSpace   string               `json:"namespace,omitempty"`
	ClusterIP   string               `json:"clusterIp,omitempty"`
	Type        string               `json:"type,omitempty"`
	Ports       []corev1.ServicePort `json:"ports,omitempty"`
	Selector    map[string]string    `json:"selector,omitempty"`
}

// DeploymentInfo ...
type DeploymentInfo struct {
	ClusterCode         string                `json:"clusterCode,omitempty"`
	NameSpace           string                `json:"namespace,omitempty"`
	Annotations         map[string]string     `json:"annotations,omitempty"`
	Labels              map[string]string     `json:"labels,omitempty"`
	Name                string                `json:"name,omitempty"`
	StartTime           string                `json:"startTime,omitempty"`
	DesiredReplicas     *int32                `json:"desiredReplicas,omitempty"`
	UpdatedReplicas     int32                 `json:"updatedReplicas,omitempty"`
	ReadyReplicas       int32                 `json:"readyReplicas,omitempty"`
	AvailableReplicas   int32                 `json:"availableReplicas,omitempty"`
	UnavailableReplicas int32                 `json:"unavailableReplicas,omitempty"`
	Group               string                `json:"group,omitempty"`
	Selector            *metav1.LabelSelector `json:"selector,omitempty"`
}

// DeploymentStatInfo ...
type DeploymentStatInfo struct {
	DesiredReplicas     int32 `json:"desiredReplicas"`
	UpdatedReplicas     int32 `json:"updatedReplicas"`
	ReadyReplicas       int32 `json:"readyReplicas"`
	AvailableReplicas   int32 `json:"availableReplicas"`
	UnavailableReplicas int32 `json:"unavailableReplicas"`
	OK                  bool  `json:"ok"`
}

type EndpointsOfCluster struct {
	ClusterName string      `json:"clusterCode,omitempty"`
	Endpoint    []*Endpoint `json:"endpoint,omitempty"`
}

type PodOfCluster struct {
	ClusterName string `json:"clusterName,omitempty"`
	Pods        []*Pod `json:"pods"`
}

// Event ...
type Event struct {
	ClusterName string `json:"clusterCode,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	ObjectKind  string `json:"objectKind,omitempty"`
	ObjectName  string `json:"objectName,omitempty"`
	Type        string `json:"type,omitempty"`
	Count       int32  `json:"count,omitempty"`
	FirstTime   string `json:"firstTime,omitempty"`
	LastTime    string `json:"lastTime,omitempty"`
	Message     string `json:"message,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type OfflinePod struct {
	Name        string            `json:"name,omitempty"`
	ClusterName string            `json:"clusterCode,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	AppName     string            `json:"appName,omitempty"`
	HostIP      string            `json:"hostIP,omitempty"`
	PodIP       string            `json:"podIP,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	OfflineTime time.Time         `json:"offlineTime,omitempty"`
}
