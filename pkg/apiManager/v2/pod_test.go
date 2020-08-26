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

var _ = Describe("test pods api", func() {
	DescribeTable("get pods by labels",
		func(appName, group, ldcLabel, namespace, symZone, podIP, phase string, expected int) {
			testServer := gin.Default()
			testServer.GET("/api/v2/cluster/:clusterCode/appPods/labels", manager.GetPodByLabels)

			w := httptest.NewRecorder()
			url := fmt.Sprintf(
				"/api/v2/cluster/all/appPods/labels?appName=%s&group=%s&ldcLabel=%s&namespace=%s&symZone=%s&podIP=%s&phase=%s",
				appName, group, ldcLabel, namespace, symZone, podIP, phase,
			)

			req, _ := http.NewRequest("GET", url, nil)
			testServer.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(expected))
			Expect(w.Body.String()).NotTo(Equal(emptyResult))
		},
		Entry("no appName return 400", "", "", "", "", "", "", "", 400),
		Entry("only appName query ok", "rdp-configuration-center-web", "", "", "", "", "", "", 200),
		Entry("only appName & sym-group query ok", "rdp-configuration-center-web", "blue", "", "", "", "", "", 200),
		Entry("only appName & ldcLabel query ok", "rdp-configuration-center-web", "", "gz01a", "", "", "", "", 200),
		Entry("only appName & namesapce query ok", "rdp-configuration-center-web", "", "", "dmall-inner", "", "", "", 200),
		Entry("only appName & sym-zone query ok", "rdp-configuration-center-web", "", "", "", "gz01", "", "", 200),
		Entry("only appName & podIP query ok", "rdp-configuration-center-web", "", "", "", "", "10.13.98.93", "", 200),
		Entry("only appName & phase query ok", "rdp-configuration-center-web", "", "", "", "", "", "Running", 200),
	)

	DescribeTable("get app group version", func(appName, group string, expected int) {
		testServer := gin.Default()
		testServer.GET("/api/v2/cluster/:clusterCode/app/group/version", manager.GetAppGroupVersion)

		w := httptest.NewRecorder()
		url := fmt.Sprintf("/api/v2/cluster/all/app/group/version?appName=%s&group=%s", appName, group)

		req, _ := http.NewRequest("GET", url, nil)
		testServer.ServeHTTP(w, req)
		fmt.Println(w.Body.String())

		Expect(w.Code).To(Equal(expected))
		Expect(w.Body.String()).NotTo(Equal(emptyResult))
	},
		Entry("no appName return 400", "", "", 400),
		Entry("only appName query ok", "rdp-configuration-center-web", "", 200),
		Entry("appName & blue group", "rdp-configuration-center-web", "blue", 200),
		Entry("appName & green group", "rdp-configuration-center-web", "green", 200),
	)
})
