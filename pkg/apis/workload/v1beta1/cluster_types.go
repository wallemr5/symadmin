package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
)

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}

// +kubebuilder:object:root=true
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ClusterSpec   `json:"spec"`
	Status            ClusterStatus `json:"status"`
}

// +kubebuilder:object:root=true
// ClusterList implements list of Cluster.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

type ClusterSpec struct {
	KubeConfig  string            `json:"kubeConfig,omitempty"`
	SymNodeName string            `json:"symNodeName"`
	Meta        map[string]string `json:"meta,omitempty"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	HelmSpec    *HelmSpec         `json:"helmSpec,omitempty"`
	AlertSpec   *AlertSpec        `json:"alertSpec,omitempty"`
	Apps        []*HelmChartSpec  `json:"apps,omitempty"`
	Pause       bool              `json:"pause"`
}

type HelmSpec struct {
	Namespace         string `json:"namespace"`
	OverrideImageSpec string `json:"overrideImageSpec,omitempty"`
	MaxHistory        int    `json:"maxHistory,omitempty"`
}

type AlertSpec struct {
	Enable bool `json:"enable"`
}

type HelmChartSpec struct {
	Name          string            `json:"name,omitempty"`
	Namespace     string            `json:"namespace,omitempty"`
	Repo          string            `json:"repo,omitempty"`
	ChartName     string            `json:"chartName,omitempty"`
	ChartVersion  string            `json:"chartVersion,omitempty"`
	OverrideValue string            `json:"overrideValue,omitempty"`
	Values        map[string]string `json:"values,omitempty"`
}

type MonitoringStatus struct {
	GrafanaEndpoint      *string `json:"grafanaEndpoint,omitempty"`
	AlertManagerEndpoint *string `json:"alertManagerEndpoint,omitempty"`
	PrometheusEndpoint   *string `json:"prometheusEndpoint,omitempty"`
}

type ClusterStatus struct {
	AppStatus        []AppHelmStatuses        `json:"appStatus,omitempty"`
	ClusterStatus    []ClusterComponentStatus `json:"clusterStatus,omitempty"`
	Version          *version.Info            `json:"version,omitempty"`
	MonitoringStatus *MonitoringStatus        `json:"monitoringStatus,omitempty"`
	NodeDetail       *NodeDetail              `json:"nodeDetail"`
}

type ClusterComponentStatus struct {
	Name       string                  `json:"name"`
	Conditions []v1.ComponentCondition `json:"conditions,omitempty"`
}

type AppHelmStatuses struct {
	Name         string `json:"name,omitempty"`
	ChartVersion string `json:"chartVersion,omitempty"`
	RlsName      string `json:"rlsName,omitempty"`
	RlsStatus    string `json:"rlsStatus,omitempty"`
	RlsVersion   int32  `json:"rlsVersion,omitempty"`
	OverrideVa   string `json:"overrideVa,omitempty"`
}

type NodeDetail struct {
	NodeStatus          []*NodeStatus   `json:"nodeStatus"`
	Capacity            v1.ResourceList `json:"capacity,omitempty"`
	Allocatable         v1.ResourceList `json:"allocatable,omitempty"`
	Requested           v1.ResourceList `json:"requested,omitempty"`
	Limits              v1.ResourceList `json:"limits,omitempty"`
	CpuUsagePercent     int32           `json:"cpuUsagePercent"`
	PodUsagePercent     int32           `json:"podUsagePercent"`
	StorageUsagePercent int32           `json:"storageUsagePercent"`
	MemoryUsagePercent  int32           `json:"memoryUsagePercent"`
}

type NodeStatus struct {
	NodeName            string          `json:"nodeName"`
	Etcd                bool            `json:"etcd"`
	ControlPlane        bool            `json:"controlPlane"`
	Worker              bool            `json:"worker"`
	Capacity            v1.ResourceList `json:"capacity,omitempty"`
	Allocatable         v1.ResourceList `json:"allocatable,omitempty"`
	Requested           v1.ResourceList `json:"requested,omitempty"`
	Limits              v1.ResourceList `json:"limits,omitempty"`
	Ready               string          `json:"ready"`
	KernelDeadlock      string          `json:"kernelDeadlock"`
	NetworkUnavailable  string          `json:"networkUnavailable"`
	OutOfDisk           string          `json:"outOfDisk"`
	MemoryPressure      string          `json:"memoryPressure"`
	DiskPressure        string          `json:"diskPressure"`
	PIDPressure         string          `json:"pidPressure"`
	CpuUsagePercent     int32           `json:"cpuUsagePercent"`
	MemoryUsagePercent  int32           `json:"memoryUsagePercent"`
	PodUsagePercent     int32           `json:"podUsagePercent"`
	StorageUsagePercent int32           `json:"storageUsagePercent"`
}
