package apiManager

import "testing"

func TestGetPodId(t *testing.T) {
	id := getPodId("amp-omp-gz01a-blue-685d5f9d87-ml7rb")
	println(id)
}
