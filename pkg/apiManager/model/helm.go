package model

type GrupEnum string

const (
	BLUE_GROUP   GrupEnum = "blue"
	GREEN_GROUP  GrupEnum = "green"
	CANARY_GROUP GrupEnum = "canary"
)

type Info struct {
	Status        Status `json:"status,omitempty"`
	Description   string `json:"description,omitempty"`
	FirstDeployed string `json:"firstDeployed,omitempty"`
	lastDeployed  string `json:"lastDeployed,omitempty"`
}

type Metadata struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Version     int32  `json:"version,omitempty"`
	ApiVersion  string `json:"apiVersion,omitempty"`
}

type Template struct {
	Name string `json:"name,omitempty"`
	Data string `json:"data,omitempty"`
}

type Chart struct {
	Metadata Metadata   `json:"metadata,omitempty"`
	Template []Template `json:"templates,omitempty"`
}

type Value struct {
	Raw string `json:"raw,omitempty"`
}

type Status struct {
	code string `json:"code,omitempty"`
}

type HelmWholeRelease struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Version   int32  `json:"version,omitempty"`
	Manifest  string `json:"manifest,omitempty"`
	Info      Info   `json:"info,omitempty"`
	Chart     Chart  `json:"chart,omitempty"`
	Value     Value  `json:"value,omitempty"`
}

type PackageInfo struct {
	TagName     string `json:"tagName,omitempty"`
	GitVersion  string `json:"gitVersion,omitempty"`
	Container   string `json:"container,omitempty"`
	TrackNo     string `json:"trackNo,omitempty"`
	HubRegistry string `json:"hubRegistry,omitempty"`
}

type HelmRelease struct {
	Cluster                string        `json:"cluster,omitempty"`
	Group                  GrupEnum      `json:"group,omitempty"`
	Name                   string        `json:"name,omitempty"`
	Version                int32         `json:"version,omitempty"`
	Description            string        `json:"description,omitempty"`
	Status                 string        `json:"status,omitempty"`
	FirstDeployedDate      string        `json:"firstDeployedDate,omitempty"`
	LastDeployedDate       string        `json:"lastDeployedDate,omitempty"`
	ReplicaCount           int32         `json:"replicaCount,omitempty"`
	PackageInfos           []PackageInfo `json:"packageInfos,omitempty"`
	IsSuccessed            bool          `json:"isSuccessed,omitempty"`
	FailedExceptionMessage string        `json:"failedExceptionMessage,omitempty"`
}
