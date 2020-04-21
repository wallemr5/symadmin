package appset

import (
	"context"
	"fmt"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
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

	if len(app.ObjectMeta.Finalizers) == 0 {
		klog.V(4).Infof("%s: finalizers is empty", req.NamespacedName)
		return nil
	}

	klog.V(4).Infof("%s: delete all AdvDeployment success, delete AppSet now", req.NamespacedName)
	app.ObjectMeta.Finalizers = utils.RemoveString(app.ObjectMeta.Finalizers, labels.ControllerFinalizersName)
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
		klog.V(4).Infof("%s: delete cluster[%s] Advdeployment success", req.NamespacedName, cluster.GetName())
		return true, nil
	}
	if apierrors.IsNotFound(err) {
		klog.Errorf("%s: delete cluster[%s] Advdeployment fail, not found", req.NamespacedName, cluster.GetName())
		return false, nil
	}

	return false, fmt.Errorf("delete cluster[%s] AdvDeployment(%s) fail:%+v", cluster.GetName(), req.NamespacedName.String(), err)
}

func (r *AppSetReconciler) DeleteUnExpectInfo(ctx context.Context, req customctrl.CustomRequest, status workloadv1beta1.AppStatus) (isChange bool, err error) {
	if status != workloadv1beta1.AppStatusRuning {
		return false, nil
	}

	app := &workloadv1beta1.AppSet{}
	if err := r.Client.Get(ctx, req.NamespacedName, app); err != nil {
		klog.Errorf("%s: get AppSet info faild, err: %+v", req.NamespacedName, err)
		return false, err
	}

	// get current info
	currentInfo := map[string]*workloadv1beta1.AdvDeployment{}
	for _, cluster := range r.DksMgr.K8sMgr.GetAll() {
		b := &workloadv1beta1.AdvDeployment{}
		err := cluster.Client.Get(ctx, req.NamespacedName, b)
		if err == nil {
			currentInfo[cluster.GetName()] = b
			continue
		}
		if apierrors.IsNotFound(err) {
			continue
		}
		return false, err
	}

	// build expect info with app
	expectInfo := map[string]struct{}{}
	for _, cluster := range app.Spec.ClusterTopology.Clusters {
		expectInfo[cluster.Name] = struct{}{}
	}

	// current equal expect
	if len(expectInfo) == len(currentInfo) {
		return false, nil
	}

	delCluster := ""
	for current := range currentInfo {
		if _, ok := expectInfo[current]; ok {
			continue
		}
		delCluster = current
		break
	}
	if delCluster == "" {
		// not exist need delete cluster
		return false, nil
	}
	klog.V(4).Infof("%s: delete unexpect info cluster:%s", req.NamespacedName, delCluster)

	client, err := r.DksMgr.K8sMgr.Get(delCluster)
	if err != nil {
		klog.Errorf("%s: delete unexpect info, get cluster[%s] client fail:%+v", req.NamespacedName, delCluster, err)
		return false, err
	}
	return deleteByCluster(ctx, client, req)
}
