package appset

import (
	"context"
	"fmt"
	"strings"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// DeleteAll delete crd handler
func (r *AppSetReconciler) DeleteAll(ctx context.Context, req customctrl.CustomRequest, app *workloadv1beta1.AppSet) error {
	// loop cluster delete advdeployment
	for _, cluster := range r.DksMgr.K8sMgr.GetAll() {
		cluster, err := r.DksMgr.K8sMgr.Get(cluster.GetName())
		if err != nil {
			return err
		}

		isChanged, err := deleteByCluster(ctx, cluster, req)
		if err != nil || isChanged {
			return err
		}
	}

	klog.V(4).Infof("%s:delete all AdvDeployment success,delete AppSet now", req.NamespacedName)
	i := -1
	for j := range app.ObjectMeta.Finalizers {
		if strings.EqualFold(app.ObjectMeta.Finalizers[j], labels.ControllerFinalizersName) {
			i = j
			break
		}
	}
	if i != -1 {
		app.ObjectMeta.Finalizers = append(app.ObjectMeta.Finalizers[:i], app.ObjectMeta.Finalizers[i+1:]...)
	}
	return r.Client.Update(ctx, app)
}

func deleteByCluster(ctx context.Context, cluster *k8smanager.Cluster, req customctrl.CustomRequest) (bool, error) {
	err := cluster.Client.Delete(ctx, &workloadv1beta1.AdvDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
	})
	if err == nil {
		klog.V(4).Infof("%s:delete cluster[%s] Advdeployment event success", req.NamespacedName, cluster.GetName())
		return true, nil
	}
	if apierrors.IsNotFound(err) {
		klog.V(4).Infof("%s:delete cluster[%s] Advdeployment event fail, not found", req.NamespacedName, cluster.GetName())
		return false, nil
	}

	klog.V(4).Infof("%s:delete cluster[%s] Advdeployment event fail:%+v", req.NamespacedName, cluster.GetName(), err)
	return false, fmt.Errorf("delete cluster:%s AdvDeployment(%s) fail:%+v", cluster.GetName(), req.NamespacedName.String(), err)
}
