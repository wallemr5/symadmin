package traefik

import (
	"fmt"

	"emperror.dev/errors"
	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/common"
	helmv3 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v3"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"k8s.io/klog"
)

type reconciler struct {
	name        string
	k           *k8smanager.Cluster
	obj         *workloadv1beta1.Cluster
	env         *helmv3.HelmEnv
	clusterType string
	urlHead     string
}

func New(k *k8smanager.Cluster, obj *workloadv1beta1.Cluster, env *helmv3.HelmEnv) common.ComponentReconciler {
	r := &reconciler{
		name: "traefik",
		k:    k,
		obj:  obj,
		env:  env,
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
			"annotations": common.GetLbServiceAnnotations(r.clusterType, r.k.Client),
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

	env, err := helmv3.NewHelmEnv(r.env, r.k.RawKubeconfig, app.Namespace, r.k.KubeCli)
	if err != nil {
		return nil, errors.Wrapf(err, "failed new helm env")
	}

	rlsName, ns, chartURL := common.BuildHelmInfo(app)
	var vaByte []byte
	if app.OverrideValue != "" {
		vaByte = []byte(app.OverrideValue)
	} else {
		valueMap := r.makeOverrideTraefikMap()
		vaByte, err = yaml.Marshal(valueMap)
		if err != nil {
			klog.Errorf("Marshal overrideValueMap err:%+v", err)
			return nil, err
		}
	}

	klog.V(4).Infof("rlsName:%s OverrideValue:\n%s", rlsName, string(vaByte))
	rls, err := helmv3.ApplyRelease(env, rlsName, chartURL, app.ChartVersion, nil, ns, vaByte, nil)
	if err != nil || rls == nil {
		return nil, err
	}

	return common.ConvertAppHelmReleasePtr(rls), nil
}
