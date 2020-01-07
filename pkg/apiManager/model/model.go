package model

import (
	corev1 "k8s.io/api/core/v1"
)

type ClusterStatus struct {
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
}

type Pod struct {
	Name         string `json:"name,omitempty"`
	NodeIp       string `json:"nodeIp,omitempty"`
	PodIp        string `json:"podIp,omitempty"`
	ImageVersion string `json:"imageVersion,omitempty"`
}

// ErrorResponse describes responses when an error occurred
type ErrorResponse struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}
type Project struct {
	NodeIp     string `json:"nodeIp,omitempty"`
	DomainName string `json:"domainName,omitempty"`
	PodIp      string `json:"podIp,omitempty"`
	PodCount   int    `json:"podCount,omitempty"`
}
type Endpoints struct {
	Name              string `json:"name,omitempty"`
	Namespace         string `json:"namespace,omitempty"`
	CreationTimestamp string `json:"creationTimes,omitempty"`
	Release           string `json:"release,omitempty"`
	ClusterName       string `json:"clusterName,omitempty"`
	Subsets           string `json:"subsets,omitempty"`
}
type NodeInfo struct {
	Name          string `json:"name,omitempty"`
	HostIp        string `json:"hostIp,omitempty"`
	OsImage       string `json:"osImage,omitempty"`
	KernelVersion string `json:"kernelVersion,omitempty"`
	PodsCount     int    `json:"podsCount,omitempty"`
	Architecture  string `json:"architecture,omitempty"`
	MemorySize    int64  `json:"memorySize,omitempty"`
	Status        string `json:"status"`
	Cpu           int64  `json:"cpu,omitempty"`
	JoinDate      string `json:"joinDate,omitempty"`
	System        string `json:"system,omitempty"`
	DockerVersion string `json:"dockerVersion,omitempty"`
}

type ServiceInfo struct {
	NameSpace string               `json:"namespace,omitempty"`
	ClusterIP string               `json:"clusterIP,omitempty"`
	Type      string               `json:"type,omitempty"`
	Ports     []corev1.ServicePort `json:"ports,omitempty"`
	Selector  map[string]string    `json:"selector,omitempty"`
}
