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

var _ = Describe("test get event", func() {
	DescribeTable("get pod event",
		func(namespace, podName, limit string, expected int) {
			testServer := gin.Default()
			testServer.GET("/api/v2/cluster/:clusterCode/namespace/:namespace/pods/:podName/event", manager.GetPodEvent)

			w := httptest.NewRecorder()
			url := fmt.Sprintf("/api/v2/cluster/all/namespace/%s/pods/%s/event?limit=%s", namespace, podName, limit)

			req, _ := http.NewRequest("GET", url, nil)
			testServer.ServeHTTP(w, req)

			fmt.Println(w.Body.String())
			Expect(w.Code).To(Equal(expected))
			Expect(w.Body.String()).NotTo(Equal(emptyResult))
		},
		Entry("incurrent namespace return empty", "", "aabb", "10", 200),
		Entry("incurrent podName return empty", "dmall-inner", "aabb", "10", 200),
		Entry("query ok", "dmall-inner", "member-server-gz01a-blue-7f6b87f4b6-88wzm", "10", 200),
		Entry("query all pod event ok", "dmall-inner", "all", "10", 200),
	)
})
