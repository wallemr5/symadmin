package model

// GroupEnum ...
type GroupEnum string

// Enum items
const (
	BlueGroup   GroupEnum = "blue"
	GreenGroup  GroupEnum = "green"
	CanaryGroup GroupEnum = "canary"
	SvcGroup    GroupEnum = "svc"
	Unkonwn     GroupEnum = "unknown"
)

// Info ...
type Info struct {
	Status        *Status `json:"status,omitempty"`
	Description   string  `json:"description,omitempty"`
	FirstDeployed string  `json:"firstDeployed,omitempty"`
	LastDeployed  string  `json:"lastDeployed,omitempty"`
}

// Metadata ...
type Metadata struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
	APIVersion  string `json:"apiVersion,omitempty"`
}

// Template ...
type Template struct {
	Name string `json:"name,omitempty"`
	Data []byte `json:"data,omitempty"`
}

// Chart ...
type Chart struct {
	Metadata  *Metadata   `json:"metadata,omitempty"`
	Templates []*Template `json:"templates,omitempty"`
	Value     *Value      `json:"values,omitempty"`
}

// Value ...
type Value struct {
	Raw string `json:"raw,omitempty"`
}

// Config ...
type Config struct {
	Raw string `json:"raw,omitempty"`
}

// Status ...
type Status struct {
	Code string `json:"code,omitempty"`
}

// HelmWholeRelease ...
type HelmWholeRelease struct {
	Name      string  `json:"name,omitempty"`
	Namespace string  `json:"namespace,omitempty"`
	Version   int32   `json:"version,omitempty"`
	Manifest  string  `json:"manifest,omitempty"`
	Info      *Info   `json:"info,omitempty"`
	Chart     *Chart  `json:"chart,omitempty"`
	Config    *Config `json:"config,omitempty"`
}

// PackageInfo ...
type PackageInfo struct {
	TagName     string `json:"tagName,omitempty"`
	GitVersion  string `json:"gitVersion,omitempty"`
	Container   string `json:"container,omitempty"`
	TrackNo     string `json:"trackNo,omitempty"`
	HubRegistry string `json:"hubRegistry,omitempty"`
}

// HelmRelease ...
type HelmRelease struct {
	Cluster                string         `json:"cluster,omitempty"`
	Group                  GroupEnum      `json:"group,omitempty"`
	Name                   string         `json:"name,omitempty"`
	Version                string         `json:"version,omitempty"`
	Description            string         `json:"description,omitempty"`
	Status                 string         `json:"status,omitempty"`
	FirstDeployedDate      string         `json:"firstDeployedDate,omitempty"`
	LastDeployedDate       string         `json:"lastDeployedDate,omitempty"`
	ReplicaCount           int32          `json:"replicaCount,omitempty"`
	PackageInfos           []*PackageInfo `json:"packageInfos,omitempty"`
	IsSuccessed            bool           `json:"isSuccessed,omitempty"`
	FailedExceptionMessage string         `json:"failedExceptionMessage,omitempty"`
}
