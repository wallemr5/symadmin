package common

import (
	"context"

	"github.com/go-logr/logr"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"

	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// kubernetes.io/hostname=10.13.133.11
var (
	LocalStorageName   = "local-storage"
	PromPvName         = "prometheus-pv"
	GrafanaPvName      = "grafana-pv"
	NodeSelectorKey    = "sym-preserve"
	NodeSelectorVa     = "monitor"
	NodeMonitorName    = "node-role.kubernetes.io/monitor"
	RepositoryHub      = "registry.cn-hangzhou.aliyuncs.com/dmall/"
	PromCrdSuffix      = "monitoring.coreos.com"
	MasterNodeLabelKey = "node-role.kubernetes.io/master"
)

// ComponentReconciler ...
type ComponentReconciler interface {
	Name() string
	Reconcile(log logr.Logger, app interface{}) (interface{}, error)
}

// GetHelmChartURL ...
func GetHelmChartURL(repo string, chartName string) string {
	if repo == "" {
		repo = "dmall"
	}

	return fmt.Sprintf("%s/%s", repo, chartName)
}

// MakeNodeAffinity ...
func MakeNodeAffinity() map[string]interface{} {
	affinity := map[string]interface{}{
		"nodeAffinity": map[string]interface{}{
			"preferredDuringSchedulingIgnoredDuringExecution": []map[string]interface{}{
				{
					"weight": 1,
					"preference": map[string]interface{}{
						"matchExpressions": []map[string]interface{}{
							{
								"key":      NodeSelectorKey,
								"operator": "In",
								"values":   []string{NodeSelectorVa},
							},
						},
					},
				},
			},
		},
	}

	return affinity
}

// MakeNodeTolerations ...
func MakeNodeTolerations() []map[string]interface{} {
	tolerations := []map[string]interface{}{
		{
			"key":      NodeSelectorKey,
			"operator": "Exists",
		},
	}

	return tolerations
}

// PreLabelsNs ...
func PreLabelsNs(k *k8smanager.Cluster, obj *workloadv1beta1.Cluster) error {
	nss := &corev1.NamespaceList{}
	err := k.Client.List(context.TODO(), &client.ListOptions{}, nss)
	if err != nil {
		klog.Errorf("list ns err: %+v", err)
		return err
	}

	var isUpdate bool
	for i := range nss.Items {
		isUpdate = false
		ns := &nss.Items[i]
		if va, ok := ns.Labels["name"]; !ok {
			isUpdate = true
		} else {
			if va != ns.Name {
				isUpdate = true
			}
		}

		if isUpdate {
			if ns.Labels == nil {
				ns.Labels = make(map[string]string)
			}
			ns.Labels["name"] = ns.Name
			err := k.Client.Update(context.TODO(), ns)
			if err != nil {
				klog.Errorf("update ns:%s name label err: %+v", ns.Name, err)
				return err
			}

			klog.Infof("update ns:%s name labels success", ns.Name)
		}
	}

	return nil
}

// PreLabelsNode ...
func PreLabelsNode(k *k8smanager.Cluster, obj *workloadv1beta1.Cluster) error {
	node := &corev1.Node{}
	err := k.Client.Get(context.TODO(), types.NamespacedName{
		Name: obj.Spec.SymNodeName,
	}, node)
	if err != nil {
		klog.Errorf("get SymNodeName:%s err: %+v", obj.Spec.SymNodeName, err)
		return err
	}

	isLabelChange := 0
	midifyLabels := node.Labels
	if keyVa, ok := node.Labels[NodeSelectorKey]; !ok && keyVa != NodeSelectorVa {
		midifyLabels[NodeSelectorKey] = NodeSelectorVa
		isLabelChange++
	}

	if _, ok := node.Labels[NodeMonitorName]; !ok {
		midifyLabels[NodeMonitorName] = "sym"
		isLabelChange++
	}

	if isLabelChange > 0 {
		node.Labels = midifyLabels
		err := k.Client.Update(context.TODO(), node)
		if err != nil {
			klog.Errorf("update labes SymNodeName:%s err: %+v", obj.Spec.SymNodeName, err)
			return err
		}

		klog.Infof("node[%s] change labels: %q success ", node.Name, node.Labels)
	}

	return nil
}

// FindComponentReconciler ...
func FindComponentReconciler(name string, cs []ComponentReconciler) (ComponentReconciler, error) {
	var t ComponentReconciler

	for _, c := range cs {
		if c.Name() == name {
			return c, nil
		}

		if c.Name() == "other" {
			t = c
		}
	}

	if t == nil {
		return nil, fmt.Errorf("not find Reconciler")
	}

	return t, nil
}
