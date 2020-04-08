package traefik

import (
	"fmt"

	"context"

	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/common"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

type reconciler struct {
	name        string
	k           *k8smanager.Cluster
	obj         *workloadv1beta1.Cluster
	hClient     *helmv2.Client
	clusterType string
	urlHead     string
}

func New(k *k8smanager.Cluster, obj *workloadv1beta1.Cluster, hClient *helmv2.Client) common.ComponentReconciler {
	r := &reconciler{
		name:    "traefik",
		k:       k,
		hClient: hClient,
		obj:     obj,
	}

	if clusterType, ok := r.obj.Spec.Meta[common.ClusterType]; ok {
		r.clusterType = clusterType
	}

	if h, ok := r.obj.Spec.Meta[common.ClusterIngressHead]; ok {
		r.urlHead = h
	}
	return r
}

func (r *reconciler) Name() string {
	return r.name
}

func (r *reconciler) makeOverrideTraefikMap() map[string]interface{} {
	serviceAnnotations := map[string]interface{}{}
	switch r.clusterType {
	case "tke":
		svc := &corev1.Service{}
		err := r.k.Client.Get(context.TODO(), types.NamespacedName{Name: "kube-user", Namespace: "default"}, svc)
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

	overrideValueMap := map[string]interface{}{
		"dashboard": map[string]interface{}{
			"enabled": true,
			"domain":  fmt.Sprintf("%s.traefik.dmall.com", r.urlHead),
			"ingress": map[string]interface{}{
				"annotations": map[string]interface{}{
					"kubernetes.io/ingress.class": "traefik",
				},
			},
		},
		"service": map[string]interface{}{
			"annotations": serviceAnnotations,
		},
		"debug": map[string]interface{}{
			"enabled": false,
		},
		"rbac": map[string]interface{}{
			"enabled": true,
		},
		"affinity":    common.MakeNodeAffinity(),
		"tolerations": common.MakeNodeTolerations(),
	}
	return overrideValueMap
}

func (r *reconciler) Reconcile(log logr.Logger, obj interface{}) (interface{}, error) {
	app, ok := obj.(*workloadv1beta1.HelmChartSpec)
	if !ok {
		return nil, fmt.Errorf("can't convert to HelmChartSpec")
	}

	log.Info("enter Reconcile", "name", app.Name)
	if app.Name == "" || app.Namespace == "" {
		return nil, fmt.Errorf("app name or namespace is empty")
	}

	rlsName, ns, chartUrl := common.BuildHelmInfo(app)
	var vaByte []byte
	var err error
	if app.OverrideValue != "" {
		vaByte = []byte(app.OverrideValue)
	} else {
		valueMap := r.makeOverrideTraefikMap()
		vaByte, err = yaml.Marshal(valueMap)
		if err != nil {
			klog.Errorf("Marshal overrideValueMap err:%+v", err)
			return nil, err
		}
		klog.Infof("rlsName:%s OverrideValue:\n%s", rlsName, string(vaByte))
	}

	rls, err := helmv2.ApplyRelease(rlsName, chartUrl, app.ChartVersion, nil, r.hClient, ns, nil, vaByte)
	if err != nil || rls == nil {
		return nil, err
	}

	return &workloadv1beta1.AppHelmStatuses{
		Name:         app.Name,
		ChartVersion: rls.GetChart().GetMetadata().GetVersion(),
		RlsName:      rls.Name,
		RlsVersion:   rls.GetVersion(),
		RlsStatus:    rls.GetInfo().GetStatus().Code.String(),
		OverrideVa:   rls.GetConfig().GetRaw(),
	}, nil
}
