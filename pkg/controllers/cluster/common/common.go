package common

import (
	"context"

	"github.com/go-logr/logr"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"

	"fmt"

	helmv3 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// kubernetes.io/hostname=10.13.133.11
var (
	LocalStorageName   = "local-storage"
	PrometheusPvName   = "prometheus-pv"
	GrafanaPvName      = "grafana-pv"
	LokiPvName         = "loki-pv"
	NodeSelectorKey    = "sym-preserve"
	NodeSelectorVa     = "monitor"
	LokiSelectorVa     = "loki-data"
	NodeMonitorName    = "node-role.kubernetes.io/monitor"
	MasterNodeLabelKey = "node-role.kubernetes.io/master"
	NodeLokiName       = "node-role.kubernetes.io/loki"

	ClusterAlert       = "clusterAlert"
	ClusterType        = "clusterType"
	ClusterIngressHead = "clusterIngressHead"
	ClusterIngressImpl = "clusterIngressImpl"
)

// ComponentReconciler ...
type ComponentReconciler interface {
	Name() string
	Reconcile(log logr.Logger, obj interface{}) (interface{}, error)
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
	err := k.Client.List(context.TODO(), nss, &client.ListOptions{})
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
func PreLabelsNode(k *k8smanager.Cluster, nodeName, nodeSelectorVa, nodeRoleName string) error {
	node := &corev1.Node{}
	err := k.Client.Get(context.TODO(), types.NamespacedName{
		Name: nodeName,
	}, node)
	if err != nil {
		klog.Errorf("get SymNodeName:%s err: %+v", nodeName, err)
		return err
	}

	isLabelChange := 0
	midifyLabels := node.Labels
	if keyVa, ok := node.Labels[NodeSelectorKey]; !ok && keyVa != nodeSelectorVa {
		midifyLabels[NodeSelectorKey] = nodeSelectorVa
		isLabelChange++
	}

	if _, ok := node.Labels[nodeRoleName]; !ok {
		midifyLabels[nodeRoleName] = "sym"
		isLabelChange++
	}

	if isLabelChange > 0 {
		node.Labels = midifyLabels
		err := k.Client.Update(context.TODO(), node)
		if err != nil {
			klog.Errorf("update labes SymNodeName:%s err: %+v", nodeName, err)
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

// BuildHelmInfo
func BuildHelmInfo(app *workloadv1beta1.HelmChartSpec) (rlsName string, ns string, chartUrl string) {
	var chartName string
	var repo string
	if app.ChartName == "" {
		chartName = app.Name
	} else {
		chartName = app.ChartName
	}

	if app.Namespace == "" {
		ns = "default"
	} else {
		ns = app.Namespace
	}

	if app.Repo == "" {
		repo = "dmall"
	} else {
		repo = app.Repo
	}

	rlsName = app.Name
	chartUrl = fmt.Sprintf("%s/%s", repo, chartName)
	return
}

func ConvertAppHelmReleasePtr(rls *helmv3.Release) *workloadv1beta1.AppHelmStatus {
	// objs := make([]*workloadv1beta1.ResourcesObject, 0, len(rls.ReleaseResources))
	// for _, res := range rls.ReleaseResources {
	// 	objs = append(objs, &workloadv1beta1.ResourcesObject{
	// 		Group: res.Group,
	// 		Kind:  res.Kind,
	// 		Name:  res.Name,
	// 	})
	// }

	ret := &workloadv1beta1.AppHelmStatus{
		Name:         rls.ChartName,
		ChartVersion: rls.Version,
		RlsName:      rls.ReleaseName,
		RlsStatus:    rls.ReleaseInfo.Status,
		RlsVersion:   rls.ReleaseVersion,
		// Resources:    objs,
	}

	return ret
}

func GetIngressImpl(lb map[string]string) string {
	if k, ok := lb[ClusterIngressImpl]; ok {
		return k
	}

	return "traefik"
}

func GetLbServiceAnnotations(clusterType string, cli client.Client) map[string]interface{} {
	serviceAnnotations := map[string]interface{}{}
	switch clusterType {
	case "tke":
		svc := &corev1.Service{}
		err := cli.Get(context.TODO(), types.NamespacedName{Name: "kube-user", Namespace: "default"}, svc)
		if err == nil && svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
			if va, ok := svc.Annotations["service.kubernetes.io/qcloud-loadbalancer-internal-subnetid"]; ok {
				klog.Infof("find tke cluster qcloud-loadbalancer-internal-subnetid[%s]", va)
				serviceAnnotations["service.kubernetes.io/qcloud-loadbalancer-internal-subnetid"] = va
			}
		}
	case "aks":
		serviceAnnotations["service.beta.kubernetes.io/azure-load-balancer-internal"] = true
	case "gke":
		serviceAnnotations["cloud.google.com/load-balancer-type"] = "Internal"
	case "ack":
	case "eks":
	}

	return serviceAnnotations
}
