package contour

import (
	"fmt"

	"emperror.dev/errors"
	"github.com/go-logr/logr"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/common"
	helmv3 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v3"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gopkg.in/yaml.v2"
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
		name: "contour",
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

func (r *reconciler) makeOverrideContour() map[string]interface{} {
	overrideValueMap := map[string]interface{}{
		"contour": map[string]interface{}{
			"resources": map[string]interface{}{
				"limits": map[string]interface{}{
					"cpu":    "200m",
					"memory": "128Mi",
				},
				"requests": map[string]interface{}{
					"cpu":    "100m",
					"memory": "64Mi",
				},
			},
			"affinity":    common.MakeNodeAffinity(),
			"tolerations": common.MakeNodeTolerations(),
		},
		"envoy": map[string]interface{}{
			"service": map[string]interface{}{
				"annotations": common.GetLbServiceAnnotations(r.clusterType, r.k.Client),
			},
			"resources": map[string]interface{}{
				"limits": map[string]interface{}{
					"cpu":    "400m",
					"memory": "256Mi",
				},
				"requests": map[string]interface{}{
					"cpu":    "200m",
					"memory": "128Mi",
				},
			},
		},
		"prometheus": map[string]interface{}{
			"serviceMonitor": map[string]interface{}{
				"enabled": true,
			},
		},
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
		valueMap := r.makeOverrideContour()
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
