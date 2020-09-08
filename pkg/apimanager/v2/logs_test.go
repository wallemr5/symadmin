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

var _ = Describe("test pod logs interface", func() {
	DescribeTable("get container logs",
		func(clusterCode, namespace, podName, container, tail string, expected int) {
			testServer := gin.Default()
			testServer.GET("/api/v2/cluster/:clusterCode/namespace/:namespace/pods/:podName/logs", manager.HandleLogs)

			w := httptest.NewRecorder()
			url := fmt.Sprintf(
				"/api/v2/cluster/%s/namespace/%s/pods/%s/logs?container=%s&tail=%s", clusterCode, namespace, podName, container, tail,
			)

			req, _ := http.NewRequest("GET", url, nil)
			testServer.ServeHTTP(w, req)

			fmt.Println(w.Body.String())
			Expect(w.Code).To(Equal(expected))
			Expect(w.Body.String()).NotTo(Equal(emptyResult))
		},
		Entry("incurrent clusterCode return 400", "aa", "b", "b", "b", "10", 400),
		Entry("incurrent namespace return 400", "tcc-gz01-bj5-test", "b", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "b", "10", 400),
		Entry("incurrent podName return 400", "tcc-gz01-bj5-test", "dmall-inner", "b", "b", "10", 400),
		Entry("incurrent container return 400", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "aabb", "10", 400),
		Entry("no container return 200", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "", "10", 200),
		Entry("incurrent tail line return 400", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "abcd-11-adb-00", "a", 400),
		Entry("empty tail line return 200", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "abcd-11-adb-00", "", 200),
		Entry("one tail line return 200", "tcc-gz01-bj5-test", "dmall-inner", "abcd-11-adb-00-gz01a-blue-955798969-jhdkt", "abcd-11-adb-00", "1", 200),
	)
})
