package v2

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("test get files", func() {
	DescribeTable("list files",
		func(clusterCode, namespace, podName, container, path string, expected int) {
			testServer := gin.Default()
			testServer.GET("/api/v2/cluster/:clusterCode/namespace/:namespace/pods/:podName/files", manager.ListFiles)

			w := httptest.NewRecorder()
			url := fmt.Sprintf("/api/v2/cluster/%s/namespace/%s/pods/%s/files?container=%s&path=%s", clusterCode, namespace, podName, container, path)

			req, _ := http.NewRequest("GET", url, nil)
			testServer.ServeHTTP(w, req)

			fmt.Println(w.Body.String())
			Expect(w.Code).To(Equal(expected))
			Expect(w.Body.String()).NotTo(Equal(emptyResult))
		},
		Entry("incurrent podName return 400", "tcc-gz01-bj5-test", "cs", "t", "bb", "", 400),
		Entry("incurrent container return 400", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "bb", "", 400),
		Entry("incurrent path return 400", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "abcd-11-adb-00", "/abc", 400),
		Entry("empty path return 200", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "abcd-11-adb-00", "", 200),
		Entry("ls /etc/yum return 200", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "abcd-11-adb-00", "/etc/yum", 200),
	)

	DescribeTable("tail files",
		func(clusterCode, namespace, podName, container, filepath, tail string, expected int) {
			testServer := gin.Default()
			testServer.GET("/api/v2/cluster/:clusterCode/namespace/:namespace/pods/:podName/tail", manager.TailFile)

			w := httptest.NewRecorder()
			url := fmt.Sprintf("/api/v2/cluster/%s/namespace/%s/pods/%s/tail?container=%s&filepath=%s&tail=%s",
				clusterCode, namespace, podName, container, filepath, tail)

			req, _ := http.NewRequest("GET", url, nil)
			testServer.ServeHTTP(w, req)

			fmt.Println(w.Body.String())
			Expect(w.Code).To(Equal(expected))
			Expect(w.Body.String()).NotTo(Equal(emptyResult))
		},
		Entry("incurrent podName return 400", "tcc-gz01-bj5-test", "cs", "t", "bb", "", "10", 400),
		Entry("incurrent container return 400", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "bb", "", "10", 400),
		Entry("incurrent filepath return 400", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "abcd-11-adb-00", "/abc.yaml", "10", 400),
		Entry("incurrent tail return 400", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "abcd-11-adb-00", "/etc/yum/version-groups.conf", "a", 400),
		Entry("empty path return 400", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "abcd-11-adb-00", "", "", 400),
		Entry("empty tail return 400", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "abcd-11-adb-00", "/etc/yum/version-groups.conf", "", 400),
		Entry("tail /etc/yum/version-groups.conf return 200", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "abcd-11-adb-00", "/etc/yum/version-groups.conf", "10", 200),
	)
})
