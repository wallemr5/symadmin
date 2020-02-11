package labels

import (
	"errors"
	"fmt"

	"k8s.io/klog"
)

// ObservedNamespace ...
var ObservedNamespace = []string{
	"default",
	"dmall-inner",
	"dmall-outer",
}

// labels
const (
	LabelCreatedBy      = "createdBy"
	LabelClusterName    = "clusterName"
	LabelLdcName        = "ldc"
	LabelAzName         = "az"
	LabelArea           = "area"
	LabelID             = "id"
	AnnotationsSpecHash = "SpecHash"
	LabelKeyZone        = "sym-available-zone"
)

// observe must labels
const (
	ObserveMustLabelClusterName = "sym-cluster-info"
	ObserveMustLabelLdcName     = "sym-ldc"
	ObserveMustLabelGroupName   = "sym-group"
	ObserveMustLabelAppName     = "app"
	ObserveMustLabelVersion     = "version"
	ObserveMustLabelDomain      = "lightningDomain0"
)

// group items
const (
	BlueGroup   = "blue"
	GreenGroup  = "green"
	CanaryGroup = "canary"
	SvcGroup    = "svc"
)

// controller name
const (
	ControllerName           = "sym-controller"
	ControllerFinalizersName = "sym-admin-finalizers"
)

// GetLabels ...
func GetLabels(clusterName string) map[string]string {
	return map[string]string{
		LabelCreatedBy:   ControllerName,
		LabelClusterName: clusterName,
	}
}

// GetCrdLabelSelector ...
func GetCrdLabelSelector() string {
	return fmt.Sprintf("%v=%v", LabelCreatedBy, ControllerName)
}

// MakeHelmReleaseFilter ...
func MakeHelmReleaseFilter(appName string) string {
	if appName == "" || appName == "all" {
		return ""
	}
	return fmt.Sprintf("^%s(-gz|-rz).*(-blue|-green|-canary|-svc)$", appName)
}

// MakeHelmReleaseFilterWithGroup ...
func MakeHelmReleaseFilterWithGroup(appName, group string) (string, error) {
	if appName == "" || appName == "all" {
		return "", nil
	}

	if group == "" {
		return fmt.Sprintf("^%s(-gz|-rz).*(-blue|-green|-canary|-svc)$", appName), nil
	}

	if x := IsValidGroup(group); !x && group != "" {
		klog.Errorf("get not valid group: %s", group)
		err := errors.New("Received incorrect group parameter")
		return "", err
	}

	return fmt.Sprintf("^%s(-gz|-rz).*(-%s)$", appName, group), nil
}

// IsValidGroup ...
func IsValidGroup(group string) bool {
	switch group {
	case
		BlueGroup,
		GreenGroup,
		CanaryGroup,
		SvcGroup:
		return true
	}
	return false
}

func GetClusterLs() map[string]string {
	return map[string]string{
		"ClusterOwner": "sym-admin",
	}
}
