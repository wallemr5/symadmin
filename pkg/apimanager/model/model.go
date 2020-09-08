package model

import (
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
	Name         string                    `json:"name,omitempty"`
	Ready        bool                      `json:"ready,omitempty"`
	RestartCount int32                     `json:"restartCount,omitempty"`
	Image        string                    `json:"image,omitempty"`
	ContainerID  string                    `json:"containerId,omitempty"`
	LastState    *ContainerStateTerminated `json:"lastState,omitempty"`
}

// ContainerStateTerminated is a terminated state of a container.
type ContainerStateTerminated struct {
	// Exit status from the last termination of the container
	ExitCode int32 `json:"exitCode" protobuf:"varint,1,opt,name=exitCode"`
	// Signal from the last termination of the container
	// +optional
	Signal int32 `json:"signal,omitempty" protobuf:"varint,2,opt,name=signal"`
	// (brief) reason from the last termination of the container
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,3,opt,name=reason"`
	// Message regarding the last termination of the container
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,4,opt,name=message"`
	// Time at which previous execution of the container started
	// +optional
	StartedAt string `json:"startedAt,omitempty" protobuf:"bytes,5,opt,name=startedAt"`
	// Time at which the container last terminated
	// +optional
	FinishedAt string `json:"finishedAt,omitempty" protobuf:"bytes,6,opt,name=finishedAt"`
	// Container's ID in the format 'docker://<container_id>'
	// +optional
	ContainerID string `json:"containerID,omitempty" protobuf:"bytes,7,opt,name=containerID"`
}

// Pod ...
type Pod struct {
	Id           string             `json:"id,omitempty"`
	Name         string             `json:"name,omitempty"`
	Namespace    string             `json:"namespace,omitempty"`
	ClusterCode  string             `json:"clusterCode,omitempty"`
	Annotations  map[string]string  `json:"annotations,omitempty"`
	HostIP       string             `json:"hostIP,omitempty"`
	Phase        corev1.PodPhase    `json:"phase,omitempty"`
	Group        string             `json:"group,omitempty"`
	Zone         string             `json:"zone,omitempty"`
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

// Endpoint ...
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

// EndpointsOfCluster ...
type EndpointsOfCluster struct {
	ClusterName string      `json:"clusterCode,omitempty"`
	Endpoint    []*Endpoint `json:"endpoint,omitempty"`
}

// PodOfCluster ...
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

// OfflinePod ...
type OfflinePod struct {
	Name        string            `json:"name,omitempty"`
	ClusterName string            `json:"clusterCode,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
	AppName     string            `json:"appName,omitempty"`
	HostIP      string            `json:"hostIP,omitempty"`
	PodIP       string            `json:"podIP,omitempty"`
	ContainerID string            `json:"containerId,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	OfflineTime string            `json:"offlineTime,omitempty"`
}
