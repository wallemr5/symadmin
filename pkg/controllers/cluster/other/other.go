package other

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

func New(k *k8smanager.Cluster, obj *workloadv1beta1.Cluster, hClient *helmv2.Client) common.ComponentReconciler {
	return &reconciler{
		name:    "other",
		k:       k,
		hClient: hClient,
		obj:     obj,
	}
}

func (r *reconciler) Name() string {
	return r.name
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

	var vaByte []byte
	var err error
	if app.OverrideValue != "" {
		vaByte = []byte(app.OverrideValue)
	} else {
		if _, ok := app.Values["sym-affinity"]; ok {
			valueMap := map[string]interface{}{
				"affinity":    common.MakeNodeAffinity(),
				"tolerations": common.MakeNodeTolerations(),
			}

			vaByte, err = yaml.Marshal(valueMap)
			if err != nil {
				klog.Errorf("app[%s] Marshal overrideValueMap err:%+v", app.Name, err)
				return nil, err
			}
		}
	}

	rlsName, ns, chartUrl := common.BuildHelmInfo(app)
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
