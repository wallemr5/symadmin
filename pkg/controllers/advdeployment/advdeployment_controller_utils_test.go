package advdeployment

import (
	"context"
	//"fmt"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestGetDeployListByByLabels(t *testing.T) {
	//fmt.Printf("Test GetDeployListByByLabels...")

	r := &AdvDeploymentReconciler{
		Name:   "AdvDeployment-controllers",
		Log:    ctrl.Log.WithName("controllers").WithName("AdvDeployment"),
		Client: fake.NewFakeClient(),
	}

	meta := v1.ObjectMeta{Name: "foo"}
	advDeploy := &workloadv1beta1.AdvDeployment{ObjectMeta: meta}

	ctx := context.Background()

	_, err := r.GetDeployListByByLabels(ctx, advDeploy)
	if err != nil {
		t.Errorf("error message:%v", err)
	}
}
