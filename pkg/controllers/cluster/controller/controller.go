package controller

import (
	"fmt"
	"strconv"

	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/common"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"k8s.io/klog"
)

type reconciler struct {
	name    string
	k       *k8smanager.Cluster
	obj     *workloadv1beta1.Cluster
	hClient *helmv2.Client
}

// New ...
func New(k *k8smanager.Cluster, obj *workloadv1beta1.Cluster, hClient *helmv2.Client) common.ComponentReconciler {
	return &reconciler{
		name:    "sym-ctl",
		k:       k,
		hClient: hClient,
		obj:     obj,
	}
}

// Name ...
func (r *reconciler) Name() string {
	return r.name
}

// Reconcile ...
func (r *reconciler) Reconcile(log logr.Logger, obj interface{}) (interface{}, error) {
	app, ok := obj.(*workloadv1beta1.HelmChartSpec)
	if !ok {
		return nil, fmt.Errorf("can't convert to HelmChartSpec")
	}

	log.Info("enter Reconcile", "name", app.Name)
	if app.Name == "" || app.Namespace == "" {
		return nil, fmt.Errorf("app name or namespace is empty")
	}

	var vaByte []byte
	var err error
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
		klog.Infof("rlsName:%s OverrideValue:\n%s", rlsName, string(vaByte))
	}

	rls, err := helmv2.ApplyRelease(rlsName, chartURL, app.ChartVersion, nil, r.hClient, ns, nil, vaByte)
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
		image["leader"] = v
	}

	overrideValueMap := map[string]interface{}{
		"affinity":    common.MakeNodeAffinity(),
		"tolerations": common.MakeNodeTolerations(),
		"image":       image,
	}

	return overrideValueMap
}
