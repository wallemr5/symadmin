package utils

import (
	"context"
	"fmt"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/common"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// tiller constants
var (
	TillerContainerName = "tiller"
	TillerDeployName    = "tiller-deploy"
	TillerNameSpace     = "kube-system"
	TillerHistoryMax    = "TILLER_HISTORY_MAX"
)

// UpgradeTiller ...
func UpgradeTiller(k *k8smanager.Cluster, opts *workloadv1beta1.Cluster) error {
	deploymentName := TillerDeployName
	obj, err := k.KubeCli.AppsV1().Deployments(opts.Spec.HelmSpec.Namespace).Get(deploymentName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	obj.Spec.Template.Spec.Containers[0].Image = opts.Spec.HelmSpec.OverrideImageSpec
	for i := range obj.Spec.Template.Spec.Containers[0].Env {
		if obj.Spec.Template.Spec.Containers[0].Env[i].Name == TillerHistoryMax {
			obj.Spec.Template.Spec.Containers[0].Env[i].Value = fmt.Sprintf("%d", opts.Spec.HelmSpec.MaxHistory)
		}
	}

	if obj.Spec.Template.Spec.Affinity == nil {
		obj.Spec.Template.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: nil,
				PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
					{
						Weight: 1,
						Preference: corev1.NodeSelectorTerm{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      common.NodeSelectorKey,
									Operator: corev1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			},
		}
	}

	if len(obj.Spec.Template.Spec.Tolerations) == 0 {
		obj.Spec.Template.Spec.Tolerations = append(obj.Spec.Template.Spec.Tolerations, corev1.Toleration{
			Key:      common.NodeSelectorKey,
			Operator: corev1.TolerationOpExists,
		})
	}

	if _, err := k.KubeCli.AppsV1().Deployments(opts.Spec.HelmSpec.Namespace).Update(obj); err != nil {
		return err
	}

	return err
}

// GetTillerDeploy ...
func GetTillerDeploy(ctx context.Context, k *k8smanager.Cluster) (*appsv1.Deployment, error) {
	var deploys appsv1.DeploymentList
	err := k.Client.List(ctx, &client.ListOptions{
		Namespace:     TillerNameSpace,
		LabelSelector: labels.SelectorFromSet(labels.Set{"name": TillerContainerName}),
	}, &deploys)
	if err != nil {
		return nil, err
	}

	if len(deploys.Items) > 0 {
		return &deploys.Items[0], nil
	}
	return nil, err
}

// InstallTiller ...
func InstallTiller(k *k8smanager.Cluster, obj *workloadv1beta1.Cluster) error {
	klog.Infof("cluster:%s starting install deploy tiller, spec:%+v", k.Name, obj.Spec.HelmSpec)
	helmInstallOption := &helmv2.Install{
		Namespace:       obj.Spec.HelmSpec.Namespace,
		Upgrade:         true,
		ServiceAccount:  TillerContainerName,
		ClusterRoleName: "cluster-admin",
		ImageSpec:       obj.Spec.HelmSpec.OverrideImageSpec,
		MaxHistory:      obj.Spec.HelmSpec.MaxHistory,
	}

	err := helmv2.PreInstallTiller(helmInstallOption, k.KubeCli)
	if err != nil {
		return err
	}
	err = helmv2.InstallTiller(helmInstallOption, k.KubeCli)
	if err != nil {
		return err
	}
	return nil
}
