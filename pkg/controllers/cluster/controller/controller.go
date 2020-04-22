package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/common"
	"gitlab.dmall.com/arch/sym-admin/pkg/helm/object"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type reconciler struct {
	name    string
	mgr     manager.Manager
	k       *k8smanager.Cluster
	obj     *workloadv1beta1.Cluster
	hClient *helmv2.Client
}

// New ...
func New(mgr manager.Manager, k *k8smanager.Cluster, obj *workloadv1beta1.Cluster, hClient *helmv2.Client) common.ComponentReconciler {
	return &reconciler{
		name:    "sym-ctl",
		mgr:     mgr,
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

	if app.ChartName == "" {
		app.ChartName = "sym-controller"
	}

	var vaByte []byte
	var err error
	rlsName, ns, chartURL := common.BuildHelmInfo(app)
	err = r.preInstallCrd(rlsName, chartURL, app.ChartVersion)
	if err != nil {
		klog.Errorf("Reconcile crd err: %v", err)
		return nil, err
	}

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

func (r *reconciler) preInstallCrd(rlsName string, chartName string, chartVersion string) error {
	chart, err := helmv2.GetRequestedChart(rlsName, chartName, chartVersion, nil, r.hClient.Env)
	if err != nil {
		return fmt.Errorf("loading chart has an error: %v", err)
	}

	for _, file := range chart.Files {
		if strings.HasPrefix(file.TypeUrl, "crds") {
			orgYaml := object.RemoveNonYAMLLines(string(file.Value))
			if orgYaml == "" {
				continue
			}
			klog.V(4).Infof("start ation name: %s ... ", file.TypeUrl)
			o, err := object.ParseYAMLToK8sObject([]byte(orgYaml))
			if err != nil {
				return errors.Wrapf(err, "Resource name: %s Failed to parse YAML to a k8s object", file.TypeUrl)
			}

			err = reconcileCrd(r.mgr, r.k, o.UnstructuredObject())
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func reconcileCrd(mgr manager.Manager, k *k8smanager.Cluster, obj *unstructured.Unstructured) error {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{}
	err := mgr.GetScheme().Convert(obj, crd, nil)
	if err != nil {
		klog.Warningf("convert crd name:%s err: %#v", obj.GetName(), err)
		return err
	}

	klog.Infof("start reconcile crd: %s", crd.Name)
	_, err = resources.Reconcile(context.TODO(), k.Client, crd, resources.DesiredStatePresent, false)
	if err != nil {
		return err
	}

	return nil
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
		image["threadiness"] = v
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
