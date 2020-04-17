package labels

import (
	"fmt"
	"testing"
)

type testAppInfo struct {
	Check bool
	Data  AppInfo
}

func TestCheckAndGetAppInfo(t *testing.T) {
	data := map[string]testAppInfo{
		"aabb-9000-gz01b-canary": testAppInfo{
			Check: true,
			Data: AppInfo{
				Name:    "aabb-9000",
				IdcName: "gz01b",
				Group:   "canary",
			},
		},
		"aabb-9000-gz01b-canary1": testAppInfo{
			Check: false,
		},
		"aabb-9000-9000-gz01b123-canary": testAppInfo{
			Check: true,
			Data: AppInfo{
				Name:    "aabb-9000-9000",
				IdcName: "gz01b123",
				Group:   "canary",
			},
		},
		"aabb-9000--gz01b-canary": testAppInfo{
			Check: true,
			Data: AppInfo{
				Name:    "aabb-9000-",
				IdcName: "gz01b",
				Group:   "canary",
			},
		},
	}

	for input, expect := range data {
		result, ok := CheckAndGetAppInfo(input)
		if ok != expect.Check {
			t.Errorf("check error: input:%s, expect:%t, result:%t", input, expect.Check, ok)
			continue
		}
		if !ok {
			continue
		}
		if result.Name != expect.Data.Name || result.IdcName != expect.Data.IdcName || result.Group != expect.Data.Group {
			t.Errorf("get app info failed, input:%s, expect:%+v, result:%+v", input, expect.Data, result)
		}
	}
}

func TestCheckEventLabel(t *testing.T) {
	name := "unidata-admin-gz01a-blue-f89757ff-n5x2l.160237aed586b459"
	fmt.Println(CheckEventLabel(name, "unidata-admin-web"))
}
