package common

import (
	"encoding/json"

	pkgLabels "gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"k8s.io/klog"
)

type HpaMetric struct {
	ResourceName string `json:"resourceName,omitempty"`
	MetricType   string `json:"metricType,omitempty"`
	MetricValue  string `json:"metricValue,omitempty"`
}

type HpaSpec struct {
	Enable      bool  `json:"enable,omitempty"`
	MaxReplicas int32 `json:"maxReplicas,omitempty"`
	MinReplicas int32 `json:"minReplicas,omitempty"`
}

func GetHpaSpecOrg(m map[string]string) string {
	if k, ok := m[pkgLabels.WorkLoadAnnotationHpa]; ok {
		return k
	}

	return ""
}

func GetHpaSpecObj(m map[string]string) *HpaSpec {
	org := GetHpaSpecOrg(m)
	if org == "" {
		return nil
	}

	hpaSpec := &HpaSpec{}
	jerr := json.Unmarshal([]byte(org), hpaSpec)
	if jerr != nil {
		klog.Errorf("unmarshal org: %s err: %v", org, jerr)
		return nil
	}
	return hpaSpec
}

func GetHpaSpecEnable(m map[string]string) bool {
	org := GetHpaSpecOrg(m)
	if org == "" {
		return false
	}

	hpaSpec := &HpaSpec{}
	jerr := json.Unmarshal([]byte(org), hpaSpec)
	if jerr != nil {
		klog.Errorf("unmarshal org: %s err: %v", org, jerr)
		return false
	}

	if !hpaSpec.Enable {
		return false
	}
	return true
}

func HpaMetricOrg(m map[string]string) string {
	if k, ok := m[pkgLabels.WorkLoadAnnotationHpaMetrics]; ok {
		return k
	}

	return ""
}

func GetHpaMetricObj(m map[string]string) []*HpaMetric {
	org := HpaMetricOrg(m)
	if org == "" {
		return nil
	}

	metrics := []*HpaMetric{}
	jerr := json.Unmarshal([]byte(org), &metrics)
	if jerr != nil {
		klog.Errorf("unmarshal org: %s err: %v", org, jerr)
		return nil
	}
	return metrics
}
