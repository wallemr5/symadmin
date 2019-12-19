package utils

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/intstr"
	"strings"
)

var ObservedNamespace = []string{
	"default",
	"dmall-inner",
	"dmall-outer",
}

const (
	ObserveMustLabelClusterName      = "sym-cluster-info"
	ObserveMustLabelAppName          = "app"
	ObserveMustLabelReleaseName      = "release"
	ObserveMustLabelLdcName          = "sym-ldc"
	ObserveMustLabelLightningDomain0 = "lightningDomain0"
	ObserveMustLabelGroupName        = "sym-group"
)

func StrPointer(s string) *string {
	return &s
}

func IntPointer(i int32) *int32 {
	return &i
}

func Int64Pointer(i int64) *int64 {
	return &i
}

func BoolPointer(b bool) *bool {
	return &b
}

func PointerToBool(flag *bool) bool {
	if flag == nil {
		return false
	}

	return *flag
}

func PointerToString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func PointerToInt32(i *int32) int32 {
	if i == nil {
		return 0
	}

	return *i
}

func IntstrPointer(i int) *intstr.IntOrString {
	is := intstr.FromInt(i)
	return &is
}

func MergeLabels(l map[string]string, l2 map[string]string) map[string]string {
	merged := make(map[string]string)
	if l == nil {
		l = make(map[string]string)
	}
	for lKey, lValue := range l {
		merged[lKey] = lValue
	}
	for lKey, lValue := range l2 {
		merged[lKey] = lValue
	}
	return merged
}

func EmptyTypedStrSlice(s ...string) []interface{} {
	ret := make([]interface{}, len(s))
	for i := 0; i < len(s); i++ {
		ret[i] = s[i]
	}
	return ret
}

func EmptyTypedFloatSlice(f ...float64) []interface{} {
	ret := make([]interface{}, len(f))
	for i := 0; i < len(f); i++ {
		ret[i] = f[i]
	}
	return ret
}

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

// SplitMetaLdcGroupKey returns ldc name and group name
func SplitMetaLdcGroupKey(key string) (ldcName, groupName string, err error) {
	parts := strings.Split(key, "-")
	switch len(parts) {
	case 1:
		// ldc only, no group
		return parts[0], "", nil
	case 2:
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("unexpected key format: %q", key)
}
