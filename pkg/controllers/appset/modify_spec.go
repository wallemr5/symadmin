package appset

import (
	"context"

	"github.com/go-logr/logr"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ModifySpec modify spec handler
func (r *AppSetReconciler) ModifySpec(logger logr.Logger, info *workloadv1beta1.AppSet, clusterTopology *workloadv1beta1.TargetCluster, req customctrl.CustomRequest) (modifyStatus bool, condition *workloadv1beta1.AppSetCondition, err error) {
	cluster, err := r.DksMgr.K8sMgr.Get(clusterTopology.Name)
	if err != nil {
		return false, nil, err
	}

	// build AdvDeployment info with AppSet TargetCluster
	new := buildAdvDeployment(info, clusterTopology)

	old := &workloadv1beta1.AdvDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: info.Name,
		},
	}
	err = cluster.Client.Get(context.TODO(), req.NamespacedName, old)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("create event", "cluster", clusterTopology.Name)
			return true, nil, cluster.Client.Create(context.TODO(), new)
		}
		return false, nil, err
	}

	if needUpdate(old, new) {
		logger.Info("update event", "cluster", clusterTopology.Name)
		return true, nil, cluster.Client.Update(context.TODO(), coverAdvDeployment(old, new))
	}

	logger.Info("do nothing", "cluster", clusterTopology.Name)
	return false, nil, nil
}

func buildAdvDeployment(info *workloadv1beta1.AppSet, clusterTopology *workloadv1beta1.TargetCluster) *workloadv1beta1.AdvDeployment {
	replica := 0
	for _, v := range clusterTopology.PodSets {
		replica += v.Replicas.IntValue()
	}

	obj := &workloadv1beta1.AdvDeployment{}
	obj.Spec.ServiceName = info.Spec.ServiceName
	obj.Spec.PodSpec = info.Spec.PodSpec
	obj.Spec.Replicas = utils.IntPointer(int32(replica))
	obj.Spec.Topology = workloadv1beta1.Topology{
		PodSets: clusterTopology.PodSets,
	}
	obj.Namespace = info.Namespace
	obj.Name = info.Name

	return obj
}

func needUpdate(old, new *workloadv1beta1.AdvDeployment) bool {
	// // TODO label
	// for k := range new.ObjectMeta.Labels {
	// 	if _, ok := new.ObjectMeta.Labels[k]; !ok {
	// 		return true
	// 	}
	// }

	if old.Namespace != new.Namespace {
		return true
	}

	return !equality.Semantic.DeepEqual(old.Spec, new.Spec)
}

func coverAdvDeployment(old, new *workloadv1beta1.AdvDeployment) *workloadv1beta1.AdvDeployment {
	// // TODO label
	// for k, v := range new.ObjectMeta.Labels {
	// 	old.ObjectMeta.Labels[k] = v
	// }

	old.Spec.ServiceName = new.Spec.ServiceName
	old.Spec.PodSpec = new.Spec.PodSpec
	old.Spec.Replicas = new.Spec.Replicas
	old.Spec.Topology = new.Spec.Topology
	old.Namespace = new.Namespace

	return old
}
