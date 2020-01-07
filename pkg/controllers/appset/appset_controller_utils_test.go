package appset

import (
	"testing"
)

type version struct {
	v1 string
	v2 string
}

func TestMergeVersion(t *testing.T) {
	r := map[string]version{
		"":            version{"", ""},
		"v1/v2/v3":    version{"v1", "v2/v3"},
		"v2/v3":       version{"", "v3/v2"},
		"v1/v2/v3/v4": version{"v2/v3", "v4/v1"},
	}

	for expect, input := range r {
		current := mergeVersion(input.v1, input.v2)
		if expect != current {
			t.Logf("input:%+v, expect:%s, current:%s", input, expect, current)
		}
	}
}
