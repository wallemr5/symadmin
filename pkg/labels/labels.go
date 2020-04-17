package labels

import (
	"errors"
	"fmt"
	"regexp"

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
func MakeHelmReleaseFilterWithGroup(appName, group, zone string) (string, error) {
	var reg string
	switch {
	case appName == "" || appName == "all":
		reg = ""
	case group == "" && zone == "":
		reg = fmt.Sprintf("^%s(-gz|-rz).*(-blue|-green|-canary|-svc)$", appName)
	case group != "" && zone == "":
		if x := IsValidGroup(group); !x && group != "" {
			klog.Errorf("get not valid group: %s", group)
			err := errors.New("Received incorrect group parameter")
			return "", err
		}
		reg = fmt.Sprintf("^%s(-gz|-rz).*(-%s)$", appName, group)
	case group == "" && zone != "":
		reg = fmt.Sprintf("^%s(-%s).*(-blue|-green|-canary|-svc)$", appName, zone)
	default:
		reg = fmt.Sprintf("^%s(-%s).*(-%s)$", appName, zone, group)
	}
	return reg, nil
}

// CheckAndGetAppInfo check name format, if throught return app info
func CheckAndGetAppInfo(name string) (info AppInfo, check bool) {
	rep, _ := regexp.Compile(`^(.*?)-(gz|rz)(.*?)-(blue|green|canary|svc)$`)
	check = rep.Match([]byte(name))
	if !check {
		return info, false
	}

	rl := rep.FindStringSubmatch(name)
	if len(rl) != 5 {
		return info, false
	}

	info.Name = rl[1]
	info.IdcName = fmt.Sprintf("%s%s", rl[2], rl[3])
	info.Group = rl[4]
	return info, true
}

// CheckEventLabel checkeventlabel
func CheckEventLabel(name string, appName string) bool {
	// name dmall-container-api-gz01a-blue-7488db8644-8zmfh
	rep, _ := regexp.Compile(fmt.Sprintf(`^(%s)-(gz|rz)(.*?)-(blue|green|canary|svc).*?$`, appName))
	return rep.Match([]byte(name))
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
