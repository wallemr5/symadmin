package appset

import (
	"context"

	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// ModifySpec modify spec handler
func (r *AppSetReconciler) ModifySpec(logger logr.Logger, app *workloadv1beta1.AppSet, clusterTopology *workloadv1beta1.TargetCluster, req customctrl.CustomRequest) (modifyStatus bool, condition *workloadv1beta1.AppSetCondition, err error) {
	cluster, err := r.DksMgr.K8sMgr.Get(clusterTopology.Name)
	if err != nil {
		return false, nil, err
	}

	// build AdvDeployment info with AppSet TargetCluster
	newObj := buildAdvDeployment(app, clusterTopology)

	_, err = applyAdvDeployment(app, cluster, newObj)
	return false, nil, err
}

func buildAdvDeployment(app *workloadv1beta1.AppSet, clusterTopology *workloadv1beta1.TargetCluster) *workloadv1beta1.AdvDeployment {
	replica := 0
	for _, v := range clusterTopology.PodSets {
		replica += v.Replicas.IntValue()
	}

	obj := &workloadv1beta1.AdvDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:       app.Name,
			Namespace:  app.Namespace,
			Labels:     makeAdvDeploymentLabel(clusterTopology, app),
			Finalizers: []string{labels.ControllerFinalizersName},
		},
	}

	if app.Spec.ServiceName != nil {
		svcName := *app.Spec.ServiceName
		obj.Spec.ServiceName = &svcName
	}

	obj.Spec.Replicas = utils.IntPointer(int32(replica))
	app.Spec.PodSpec.DeepCopyInto(&obj.Spec.PodSpec)

	for i := range clusterTopology.PodSets {
		podSet := clusterTopology.PodSets[i].DeepCopy()
		if podSet.RawValues == "" {
			podSet.RawValues = makeHelmOverrideValus(podSet.Name, clusterTopology, app)
		}
		obj.Spec.Topology.PodSets = append(obj.Spec.Topology.PodSets, *podSet)
	}
	return obj
}

func applyAdvDeployment(app *workloadv1beta1.AppSet, cluster *k8smanager.Cluster, advDeploy *workloadv1beta1.AdvDeployment) (*workloadv1beta1.AdvDeployment, error) {
	obj := &workloadv1beta1.AdvDeployment{}
	err := cluster.Client.Get(context.TODO(), types.NamespacedName{
		Name:      app.Name,
		Namespace: app.Namespace,
	}, obj)

	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Infof("create spec cluster:%s ", cluster.Name)
			time := metav1.Now()
			advDeploy.Status.StartTime = &time
			err = cluster.Client.Create(context.TODO(), advDeploy)
			if err != nil {
				return nil, errors.Wrapf(err, "cluster:%s create advDeploy", cluster.Name)
			}
			return advDeploy, nil
		}
		return nil, err
	}

	var isChanged int
	isEqual := equality.Semantic.DeepEqual(obj.Spec, advDeploy.Spec)
	if !isEqual {
		isChanged++
	}

	isEqual = equality.Semantic.DeepEqual(obj.ObjectMeta.Labels, advDeploy.ObjectMeta.Labels)
	if !isEqual {
		isChanged++
	}

	if isChanged > 0 {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			time := metav1.Now()
			advDeploy.Spec.DeepCopyInto(&obj.Spec)
			obj.Labels = advDeploy.ObjectMeta.Labels
			obj.Status.LastUpdateTime = &time
			updateErr := cluster.Client.Update(context.TODO(), obj)
			if updateErr == nil {
				klog.V(3).Infof("advDeploy: [%s/%s] updated successfully", cluster.Name, advDeploy.Name)
				return nil
			}

			getErr := cluster.Client.Get(context.TODO(), types.NamespacedName{
				Name:      app.Name,
				Namespace: app.Namespace,
			}, obj)

			if getErr != nil {
				utilruntime.HandleError(fmt.Errorf("getting updated advDeploy: [%s/%s] err: %v", cluster.Name, advDeploy.Name, err))
			}
			return updateErr
		})
		return nil, err
	}

	return obj, nil
}
