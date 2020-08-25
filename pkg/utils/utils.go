package utils

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"reflect"
	"unsafe"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
)

// StrPointer ...
func StrPointer(s string) *string {
	return &s
}

// IntPointer ...
func IntPointer(i int32) *int32 {
	return &i
}

// Int64Pointer ...
func Int64Pointer(i int64) *int64 {
	return &i
}

// BoolPointer ...
func BoolPointer(b bool) *bool {
	return &b
}

// PointerToBool ...
func PointerToBool(flag *bool) bool {
	if flag == nil {
		return false
	}

	return *flag
}

// PointerToString ...
func PointerToString(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

// PointerToInt32 ...
func PointerToInt32(i *int32) int32 {
	if i == nil {
		return 0
	}

	return *i
}

// IntstrPointer ...
func IntstrPointer(i int) *intstr.IntOrString {
	is := intstr.FromInt(i)
	return &is
}

// MergeLabels ...
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

// EmptyTypedStrSlice ...
func EmptyTypedStrSlice(s ...string) []interface{} {
	ret := make([]interface{}, len(s))
	for i := 0; i < len(s); i++ {
		ret[i] = s[i]
	}
	return ret
}

// EmptyTypedFloatSlice ...
func EmptyTypedFloatSlice(f ...float64) []interface{} {
	ret := make([]interface{}, len(f))
	for i := 0; i < len(f); i++ {
		ret[i] = f[i]
	}
	return ret
}

// ContainsString ...
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// RemoveString ...
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

// ToClusterCrName ...
func ToClusterCrName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "_", "-"))
}

// FillImageVersion ...
func FillImageVersion(name string, podSpec *corev1.PodSpec) string {
	if podSpec == nil {
		return ""
	}

	for i := range podSpec.Containers {
		c := &podSpec.Containers[i]
		if strings.HasPrefix(c.Name, name) {
			fullName := strings.Split(c.Image, ":")
			if len(fullName) > 1 {
				return fullName[1]
			}
		}
	}

	return ""
}

// FillDuplicatedVersion ...
func FillDuplicatedVersion(infos []*workloadv1beta1.PodSetStatusInfo) string {
	found := make(map[string]bool)
	var foundSet []string
	for i := range infos {
		if infos[i].Version != "" {
			found[infos[i].Version] = true
		}
	}

	for k := range found {
		foundSet = append(foundSet, k)
	}

	sort.Strings(foundSet)

	return strings.Join(foundSet, "/")
}

// String2bytes ...
func String2bytes(s string) []byte {
	stringHeader := (*reflect.StringHeader)(unsafe.Pointer(&s))

	bh := reflect.SliceHeader{
		Data: stringHeader.Data,
		Len:  stringHeader.Len,
		Cap:  stringHeader.Len,
	}

	return *(*[]byte)(unsafe.Pointer(&bh))
}

// Bytes2string ...
func Bytes2string(b []byte) string {
	sliceHeader := (*reflect.SliceHeader)(unsafe.Pointer(&b))

	sh := reflect.StringHeader{
		Data: sliceHeader.Data,
		Len:  sliceHeader.Len,
	}

	return *(*string)(unsafe.Pointer(&sh))
}

// FormatTime ...
func FormatTime(dt string) string {
	loc, _ := time.LoadLocation("Asia/Chongqing")
	result, err := time.ParseInLocation("2006-01-02 15:04:05 -0700 MST", dt, loc)
	if err != nil {
		klog.Errorf("time parse error: %v", err)
		return ""
	}
	return result.Format("2006-01-02 15:04:05")
}
