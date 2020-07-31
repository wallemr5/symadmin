package labels

import (
	"fmt"
	"regexp"
)

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
	ObserveMustLabelClusterName      = "sym-cluster-info"
	ObserveMustLabelAppName          = "app"
	ObserveMustLabelVersion          = "version"
	ObserveMustLabelReleaseName      = "release"
	ObserveMustLabelLdcName          = "sym-ldc"
	ObserveMustLabelLightningDomain0 = "lightningDomain0"
	ObserveMustLabelGroupName        = "sym-group"
)

const (
	ClusterAnnotationMonitor     = "k8s.io/monitor"
	WorkLoadAnnotationHpa        = "hpa.autoscaling.dmall.com/Hpa"
	WorkLoadAnnotationHpaMetrics = "hpa.autoscaling.dmall.com/Metrics"
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

// ObservedNamespace ...
var ObservedNamespace = []string{
	"default",
	"dmall-inner",
	"dmall-outer",
}

var AnnotationsKnownKey = []string{
	ClusterAnnotationMonitor,
	WorkLoadAnnotationHpa,
	WorkLoadAnnotationHpaMetrics,
}

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

func GetAnnotationKey(annotation map[string]string, key string) string {
	if k, ok := annotation[key]; ok {
		return k
	}

	return ""
}
