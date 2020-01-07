package appset

import (
	"fmt"
	"testing"
)

func TestMergeVersion(t *testing.T) {
	v1 := "v1/v2"
	v2 := "v2/v3"
	fmt.Println(mergeVersion(v1, v2))
}
