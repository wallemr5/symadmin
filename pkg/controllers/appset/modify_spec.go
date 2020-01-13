package appset

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

// ModifySpec modify spec handler
func (r *AppSetReconciler) ModifySpec(ctx context.Context, req customctrl.CustomRequest) (isChanged bool, err error) {
	app := &workloadv1beta1.AppSet{}
	if err := r.Client.Get(ctx, req.NamespacedName, app); err != nil {
		klog.Errorf("%s: applyStatus get AppSet info fail: %+v", req.NamespacedName, err)
		return false, err
	}

	for _, v := range app.Spec.ClusterTopology.Clusters {
		cluster, err := r.DksMgr.K8sMgr.Get(v.Name)
		if err != nil {
			r.recorder.Eventf(app.DeepCopy(), corev1.EventTypeWarning, "ClusterOffline", "Get cluster[%s] err:%+v.", v.Name, err)
			return false, err
		}

		newObj := buildAdvDeployment(app, v, r.DksMgr.Opt.Debug)
		_, isChanged, err = applyAdvDeployment(ctx, cluster, req, app, newObj)
		if err != nil {
			return false, err
		}
		if isChanged {
			return true, nil
		}
	}

	return false, nil
}

func buildAdvDeployment(app *workloadv1beta1.AppSet, clusterTopology *workloadv1beta1.TargetCluster, debug bool) *workloadv1beta1.AdvDeployment {
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

	for _, set := range clusterTopology.PodSets {
		podSet := set.DeepCopy()
		if debug && podSet.RawValues == "" {
			podSet.RawValues = makeHelmOverrideValus(podSet.Name, clusterTopology, app)
		}
		obj.Spec.Topology.PodSets = append(obj.Spec.Topology.PodSets, podSet)
	}
	return obj
}

func applyAdvDeployment(ctx context.Context, cluster *k8smanager.Cluster, req customctrl.CustomRequest, app *workloadv1beta1.AppSet, advDeploy *workloadv1beta1.AdvDeployment) (adv *workloadv1beta1.AdvDeployment, isChanged bool, err error) {
	obj := &workloadv1beta1.AdvDeployment{}
	err = cluster.Client.Get(ctx, types.NamespacedName{
		Name:      app.Name,
		Namespace: app.Namespace,
	}, obj)

	if err != nil {
		if apierrors.IsNotFound(err) {
			advDeploy.Status.AggrStatus.Status = workloadv1beta1.AppStatusInstalling
			err = cluster.Client.Create(ctx, advDeploy)
			if err != nil {
				return nil, false, errors.Wrapf(err, "cluster:%s create advDeploy", cluster.GetName())
			}
			klog.V(4).Infof("%s: cluster[%s] create AdvDeployment spec info", req.NamespacedName, cluster.GetName())
			return advDeploy, true, nil
		}
		return nil, false, err
	}

	if compare(obj, advDeploy) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			time := metav1.Now()
			advDeploy.Spec.DeepCopyInto(&obj.Spec)
			obj.Labels = advDeploy.ObjectMeta.Labels
			obj.Status.LastUpdateTime = &time
			obj.Status.AggrStatus.Status = workloadv1beta1.AppStatusInstalling
			updateErr := cluster.Client.Update(ctx, obj)
			if updateErr == nil {
				klog.V(4).Infof("%s: cluster[%s] AdvDeployment update successfully", req.NamespacedName, cluster.GetName())
				return nil
			}

			getErr := cluster.Client.Get(ctx, types.NamespacedName{
				Name:      app.Name,
				Namespace: app.Namespace,
			}, obj)

			if getErr != nil {
				utilruntime.HandleError(fmt.Errorf("getting updated advDeploy: [%s/%s] err: %v", cluster.Name, advDeploy.Name, err))
			}
			return updateErr
		})
		return nil, true, err
	}

	return obj, false, nil
}

func compare(new, old *workloadv1beta1.AdvDeployment) bool {
	if !equality.Semantic.DeepEqual(new.Spec, old.Spec) {
		return true
	}

	if !equality.Semantic.DeepEqual(new.ObjectMeta.Labels, old.ObjectMeta.Labels) {
		return true
	}
	return false
}
