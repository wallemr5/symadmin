package controller

import (
	"fmt"
	"strconv"

	"emperror.dev/errors"
	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/common"
	helmv3 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v3"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type reconciler struct {
	name string
	mgr  manager.Manager
	k    *k8smanager.Cluster
	obj  *workloadv1beta1.Cluster
	env  *helmv3.HelmEnv
}

// New ...
func New(mgr manager.Manager, k *k8smanager.Cluster, obj *workloadv1beta1.Cluster, env *helmv3.HelmEnv) common.ComponentReconciler {
	return &reconciler{
		name: "sym-ctl",
		mgr:  mgr,
		k:    k,
		obj:  obj,
		env:  env,
	}
}

// Name ...
func (r *reconciler) Name() string {
	return r.name
}

func makeOverrideSymCtl(app *workloadv1beta1.HelmChartSpec) map[string]interface{} {
	image := make(map[string]interface{})
	image["pullPolicy"] = "IfNotPresent"
	if v, ok := app.Values["master"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			klog.Errorf("Parse bool error: %+v", err)
		} else {
			image["master"] = b
		}
	}
	if v, ok := app.Values["worker"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			klog.Errorf("Parse bool error: %+v", err)
		} else {
			image["worker"] = b
		}
	}
	if v, ok := app.Values["cluster"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			klog.Errorf("Parse bool error: %+v", err)
		} else {
			image["cluster"] = b
		}
	}
	if v, ok := app.Values["offlinepod"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			klog.Errorf("Parse bool error: %+v", err)
		} else {
			image["offlinepod"] = b
		}
	}
	if v, ok := app.Values["reCreate"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			klog.Errorf("Parse bool error: %+v", err)
		} else {
			image["reCreate"] = b
		}
	}
	if v, ok := app.Values["leader"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			klog.Errorf("Parse bool error: %+v", err)
		} else {
			image["leader"] = b
		}
	}
	if v, ok := app.Values["oldCluster"]; ok {
		b, err := strconv.ParseBool(v)
		if err != nil {
			klog.Errorf("Parse bool error: %+v", err)
		} else {
			image["oldCluster"] = b
		}
	}
	if v, ok := app.Values["threadiness"]; ok {
		image["threadiness"] = v
	}
	if v, ok := app.Values["repos"]; ok {
		image["repos"] = v
	}
	if v, ok := app.Values["tag"]; ok {
		image["tag"] = v
	}

	overrideValueMap := map[string]interface{}{
		"affinity":    common.MakeNodeAffinity(),
		"tolerations": common.MakeNodeTolerations(),
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

	if app.ChartName == "" {
		app.ChartName = "sym-controller"
	}

	var vaByte []byte
	rlsName, ns, chartURL := common.BuildHelmInfo(app)
	if app.OverrideValue != "" {
		vaByte = []byte(app.OverrideValue)
	} else {
		valueMap := makeOverrideSymCtl(app)
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
