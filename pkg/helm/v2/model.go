package v2

import "time"

// GetReleaseResponse describes the details of a helm release
type GetReleaseResponse struct {
	ReleaseName  string                 `json:"releaseName"`
	Chart        string                 `json:"chart"`
	ChartName    string                 `json:"chartName"`
	ChartVersion string                 `json:"chartVersion"`
	Namespace    string                 `json:"namespace"`
	Version      int32                  `json:"version"`
	Status       string                 `json:"status"`
	Description  string                 `json:"description"`
	CreatedAt    time.Time              `json:"createdAt,omitempty"`
	Updated      time.Time              `json:"updatedAt,omitempty"`
	Notes        string                 `json:"notes"`
	Values       map[string]interface{} `json:"values"`
	Manifest     []ReleaseResource      `json:"Manifest"`
}

// Describes a K8s resource
type ReleaseResource struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}
