package api

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
	name string
	k    *k8smanager.Cluster
	obj  *workloadv1beta1.Cluster
	env  *helmv3.HelmEnv
}

// New ...
func New(k *k8smanager.Cluster, obj *workloadv1beta1.Cluster, env *helmv3.HelmEnv) common.ComponentReconciler {
	return &reconciler{
		name: "sym-api",
		k:    k,
		obj:  obj,
		env:  env,
	}
}

// Name ...
func (r *reconciler) Name() string {
	return r.name
}

func makeOverrideSymAPI(app *workloadv1beta1.HelmChartSpec, obj *workloadv1beta1.Cluster) map[string]interface{} {
	ingress := make(map[string]interface{})
	if v, ok := app.Values["hosts"]; ok {
		ingress["enabled"] = true
		ingress["annotations"] = map[string]interface{}{
			"kubernetes.io/ingress.class": common.GetIngressImpl(obj.Spec.Meta),
		}
		ingress["hosts"] = []map[string]interface{}{
			{
				"host":  v,
				"paths": []string{"/"},
			},
		}
	}

	image := make(map[string]interface{})
	image["pullPolicy"] = "IfNotPresent"
	if v, ok := app.Values["tag"]; ok {
		image["tag"] = v
	}
	if v, ok := app.Values["repository"]; ok {
		image["repository"] = v
	}

	overrideValueMap := map[string]interface{}{
		"affinity":    common.MakeNodeAffinity(),
		"tolerations": common.MakeNodeTolerations(),
		"ingress":     ingress,
		"image":       image,
	}

	return overrideValueMap
}

// Reconcile ...
func (r *reconciler) Reconcile(log logr.Logger, obj interface{}) (interface{}, error) {
	app, ok := obj.(*workloadv1beta1.HelmChartSpec)
	if !ok {
		return nil, fmt.Errorf("can't convert to HelmChartSpec")
	}

	env, err := helmv3.NewHelmEnv(r.env, r.k.RawKubeconfig, app.Namespace, r.k.KubeCli)
	if err != nil {
		return nil, errors.Wrapf(err, "failed new helm env")
	}

	log.Info("enter Reconcile", "name", app.Name)
	if app.Name == "" || app.Namespace == "" {
		return nil, fmt.Errorf("app name or namespace is empty")
	}

	var vaByte []byte
	rlsName, ns, chartURL := common.BuildHelmInfo(app)
	if app.OverrideValue != "" {
		vaByte = []byte(app.OverrideValue)
	} else {
		valueMap := makeOverrideSymAPI(app, r.obj)
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
