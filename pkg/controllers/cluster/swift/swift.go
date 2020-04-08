package swift

import (
	"fmt"

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
		name:    "swift",
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
		valueMap := makeOverrideSwift(app)

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

func makeOverrideSwift(app *workloadv1beta1.HelmChartSpec) map[string]interface{} {
	ingress := make(map[string]interface{})
	if va, ok := app.Values["swift-ing"]; ok {
		ingress["enabled"] = true
		ingress["annotations"] = map[string]interface{}{
			"kubernetes.io/ingress.class": "traefik",
		}
		ingress["domain"] = va
	} else {
		ingress["enabled"] = false
	}

	overrideValueMap := map[string]interface{}{
		"affinity":    common.MakeNodeAffinity(),
		"tolerations": common.MakeNodeTolerations(),
		"ingress":     ingress,
	}

	return overrideValueMap
}
