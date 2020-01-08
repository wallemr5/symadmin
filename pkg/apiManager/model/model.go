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
	Name         string `json:"name,omitempty"`
	Ready        bool   `json:"ready,omitempty"`
	RestartCount int32  `json:"restartCount,omitempty"`
	Image        string `json:"image,omitempty"`
	ContainerID  string `json:"containerId,omitempty"`
}

// Pod ...
type Pod struct {
	Name            string             `json:"name,omitempty"`
	NodeIP          string             `json:"nodeIp,omitempty"`
	PodIP           string             `json:"podIp,omitempty"`
	ImageVersion    string             `json:"imageVersion,omitempty"`
	StartTime       string             `json:"startTime,omitempty"`
	ContainerStatus []*ContainerStatus `json:"containerStatus,omitempty"`
}

// ErrorResponse describes responses when an error occurred
type ErrorResponse struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Project ...
type Project struct {
	DomainName string `json:"domainName,omitempty"`
	PodIP      string `json:"podIp,omitempty"`
}

// NodeProjects ...
type NodeProjects struct {
	NodeName string     `json:"nodeName,omitempty"`
	NodeIP   string     `json:"nodeIp,omitempty"`
	PodCount int        `json:"podCount,omitempty"`
	Projects []*Project `json:"projects,omitempty"`
}

// Endpoints ...
type Endpoint struct {
	Name              string `json:"name,omitempty"`
	Namespace         string `json:"namespace,omitempty"`
	CreationTimestamp string `json:"creationTimes,omitempty"`
	Release           string `json:"release,omitempty"`
	ClusterName       string `json:"clusterName,omitempty"`
	Subsets           string `json:"subsets,omitempty"`
}

// NodeInfo ...
type NodeInfo struct {
	Name          string `json:"name,omitempty"`
	HostIP        string `json:"hostIp,omitempty"`
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
	NameSpace string               `json:"namespace,omitempty"`
	ClusterIP string               `json:"clusterIp,omitempty"`
	Type      string               `json:"type,omitempty"`
	Ports     []corev1.ServicePort `json:"ports,omitempty"`
	Selector  map[string]string    `json:"selector,omitempty"`
}

// DeploymentInfo ...
type DeploymentInfo struct {
	Cluster             string                `json:"cluster,omitempty"`
	NameSpace           string                `json:"namespace,omitempty"`
	Name                string                `json:"name,omitempty"`
	DesiredReplicas     *int32                `json:"desiredReplicas,omitempty"`
	UpdatedReplicas     int32                 `json:"updatedReplicas,omitempty"`
	ReadyReplicas       int32                 `json:"readyReplicas,omitempty"`
	AvailableReplicas   int32                 `json:"availableReplicas,omitempty"`
	UnavailableReplicas int32                 `json:"unavailableReplicas,omitempty"`
	Group               string                `json:"group,omitempty"`
	Selector            *metav1.LabelSelector `json:"selector,omitempty"`
	CreationTimestamp   metav1.Time           `json:"creationTimestamp,omitempty"`
}

type EndpointsOfCluster struct {
	ClusterName string     `json:"clusterName,omitempty"`
	Endpoint    []Endpoint `json:"endpoint,omitempty"`
}
