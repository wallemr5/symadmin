package labels

import (
	"fmt"
)

var ObservedNamespace = []string{
	"default",
	"dmall-inner",
	"dmall-outer",
}

const (
	LabelCreatedBy      = "createdBy"
	LabelClusterName    = "clusterName"
	LabelLdcName        = "ldc"
	LabelAzName         = "az"
	LabelArea           = "area"
	LabelId             = "id"
	AnnotationsSpecHash = "SpecHash"
)

const (
	ObserveMustLabelClusterName = "sym-cluster-info"
	ObserveMustLabelLdcName     = "sym-ldc"
	ObserveMustLabelAppName     = "app"
	ObserveMustLabelVersion     = "version"
)

const (
	ControllerName = "sym-controller"
)

func GetLabels(clusterName string) map[string]string {
	return map[string]string{
		LabelCreatedBy:   ControllerName,
		LabelClusterName: clusterName,
	}
}

func GetCrdLabelSelector() string {
	return fmt.Sprintf("%v=%v", LabelCreatedBy, ControllerName)
}

func MakeHelmReleaseFilter(appName string) string {
	if appName == "" || appName == "all" {
		return ""
	}
	return fmt.Sprintf("^%s(-gz|-rz).*(-blue|-green)$", appName)
}
