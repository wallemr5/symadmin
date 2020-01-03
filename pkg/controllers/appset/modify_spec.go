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
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
)

type workflow string

const (
	UpdateStatus workflow = "Update"
	MiddleStatus workflow = "Middle"
	DeleteStatus workflow = "Delete"
	UnknowStatus workflow = "Unknow"
)

// ModifySpec modify spec handler
func (r *AppSetReconciler) ModifySpec(ctx context.Context, app *workloadv1beta1.AppSet, req customctrl.CustomRequest) (isChanged bool, err error) {
	wf, err := getCurrentDetailChoiceWorkflow(ctx, r.DksMgr.K8sMgr, app, req)
	if err != nil {
		return false, err
	}

	switch wf {
	case UpdateStatus:
		return UpdateWorkFlow(ctx, r.DksMgr.K8sMgr, app, req)
	case MiddleStatus:
		return false, nil
	case DeleteStatus:
		return false, nil
	default:
		return false, fmt.Errorf("update spec unknow workflow:%s", wf)
	}
}

func getCurrentDetailChoiceWorkflow(ctx context.Context, dksManger *k8smanager.ClusterManager, app *workloadv1beta1.AppSet, req customctrl.CustomRequest) (workflow, error) {
	currentInfo := map[string]*workloadv1beta1.AdvDeployment{}
	for _, cluster := range dksManger.GetAll() {
		b := &workloadv1beta1.AdvDeployment{}
		err := cluster.Client.Get(ctx, req.NamespacedName, b)
		if err == nil {
			currentInfo[cluster.GetName()] = b
			continue
		}
		if apierrors.IsNotFound(err) {
			continue
		}
		return UnknowStatus, err
	}
	expectInfo := map[string]struct{}{}
	for _, cluster := range app.Spec.ClusterTopology.Clusters {
		expectInfo[cluster.Name] = struct{}{}
	}

	delQueue := currentInfo
	addQueue := map[string]struct{}{}
	for exp := range expectInfo {
		if _, ok := currentInfo[exp]; ok {
			delete(delQueue, exp)
		} else {
			addQueue[exp] = struct{}{}
		}
	}

	if len(addQueue) > 0 {
		return UpdateStatus, nil
	}
	if len(delQueue) == 0 {
		return UpdateStatus, nil
	}

	// TODO: judge all current status complete, and delete unexpect cluster info
	for _, info := range currentInfo {
		if info.Status.AggrStatus.Status != "Running" {
			return MiddleStatus, nil
		}
	}
	return DeleteStatus, nil
}

func UpdateWorkFlow(ctx context.Context, dksManger *k8smanager.ClusterManager, app *workloadv1beta1.AppSet, req customctrl.CustomRequest) (isChanged bool, err error) {
	for _, v := range app.Spec.ClusterTopology.Clusters {
		cluster, err := dksManger.Get(v.Name)
		if err != nil {
			return false, err
		}

		newObj := buildAdvDeployment(app, v)
		_, isChanged, err = applyAdvDeployment(ctx, cluster, app, newObj)
		if err != nil {
			return false, err
		}
		if isChanged {
			return true, nil
		}
	}

	return false, nil
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

	for _, set := range clusterTopology.PodSets {
		podSet := set.DeepCopy()
		if podSet.RawValues == "" && app.Name != "nginx" {
			podSet.RawValues = makeHelmOverrideValus(podSet.Name, clusterTopology, app)
		}
		obj.Spec.Topology.PodSets = append(obj.Spec.Topology.PodSets, podSet)
	}
	return obj
}

func applyAdvDeployment(ctx context.Context, cluster *k8smanager.Cluster, app *workloadv1beta1.AppSet, advDeploy *workloadv1beta1.AdvDeployment) (adv *workloadv1beta1.AdvDeployment, isChanged bool, err error) {
	logger := utils.GetCtxLogger(ctx)

	obj := &workloadv1beta1.AdvDeployment{}
	err = cluster.Client.Get(ctx, types.NamespacedName{
		Name:      app.Name,
		Namespace: app.Namespace,
	}, obj)

	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("create spec cluster", "cluster", cluster.Name)
			time := metav1.Now()
			advDeploy.Status.StartTime = &time
			err = cluster.Client.Create(ctx, advDeploy)
			if err != nil {
				return nil, false, errors.Wrapf(err, "cluster:%s create advDeploy", cluster.Name)
			}
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
			updateErr := cluster.Client.Update(ctx, obj)
			if updateErr == nil {
				logger.Info("advDeploy updated successfully", "cluster", cluster.Name)
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
