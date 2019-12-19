package patch

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestAnnotationRemovedWhenEmpty(t *testing.T) {
	u := unstructured.Unstructured{}
	u.SetAnnotations(map[string]string{
		LastAppliedConfig: "{}",
	})
	modified, err := DefaultAnnotator.GetModifiedConfiguration(&u, false)
	if err != nil {
		t.Fatal(err)
	}
	if "{\"metadata\":{}}" != string(modified) {
		t.Fatalf("Expected {\"metadata\":{} got %s", string(modified))
	}
}
