package model

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
