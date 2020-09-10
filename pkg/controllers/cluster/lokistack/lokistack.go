package lokistack

import (
	"context"
	"fmt"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"emperror.dev/errors"
	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/common"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/utils"
	helmv3 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v3"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	pkgLabels "gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"k8s.io/klog"
)

type reconciler struct {
	name        string
	k           *k8smanager.Cluster
	obj         *workloadv1beta1.Cluster
	env         *helmv3.HelmEnv
	ingressImpl string
	urlHead     string
}

func New(k *k8smanager.Cluster, obj *workloadv1beta1.Cluster, env *helmv3.HelmEnv) common.ComponentReconciler {
	r := &reconciler{
		name: "loki-stack",
		k:    k,
		obj:  obj,
		env:  env,
	}
	r.ingressImpl = "contour"
	if h, ok := r.obj.Spec.Meta[common.ClusterIngressHead]; ok {
		r.urlHead = h
	}
	return r
}

func (r *reconciler) Name() string {
	return r.name
}

func getLokiStorageSize(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["lpv-size"]; ok {
		return va
	}
	return "30Gi"
}

func getLokiPvPath(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["lpv-path"]; ok {
		return va
	}
	return "/root/loki-data"
}

func preInstallLpv(k *k8smanager.Cluster, app *workloadv1beta1.HelmChartSpec, c *workloadv1beta1.Cluster) error {
	reclaimPolicy := corev1.PersistentVolumeReclaimDelete
	volumeBindingMode := storagev1.VolumeBindingWaitForFirstConsumer
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   common.LocalStorageName,
			Labels: pkgLabels.GetLabels(k.Name),
		},
		Provisioner:       "kubernetes.io/no-provisioner",
		ReclaimPolicy:     &reclaimPolicy,
		VolumeBindingMode: &volumeBindingMode,
	}

	klog.Infof("start reconcile StorageClasses: %s", sc.Name)
	_, err := resources.Reconcile(context.TODO(), k.Client, sc, resources.Option{})
	if err != nil {
		klog.Infof("start reconcile sc: %s", err)
		return err
	}

	lokiPath := getLokiPvPath(app)
	if v := pkgLabels.GetAnnotationKey(c.Annotations, pkgLabels.ClusterAnnotationLoki); v == "" {
		err = utils.ApplyLauncherPod(k.KubeCli, "default", c.Spec.LokiNodeName, lokiPath)
		if err != nil {
			klog.Errorf("ApplyLauncherPod node: %s path: %s err: %v", c.Spec.LokiNodeName, lokiPath, err)
			return err
		}

		if c.Annotations == nil {
			c.Annotations = make(map[string]string)
		}

		c.Annotations[pkgLabels.ClusterAnnotationLoki] = fmt.Sprintf("{node: %s, lokiDateDir: %s}", c.Spec.LokiNodeName, lokiPath)
	}

	lokiPv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   common.LokiPvName,
			Labels: pkgLabels.GetLabels(k.Name),
		},
		Spec: corev1.PersistentVolumeSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Capacity: corev1.ResourceList{
				"storage": resource.MustParse(getLokiStorageSize(app)),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				Local: &corev1.LocalVolumeSource{
					Path: lokiPath,
				},
			},
			PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimRetain,
			StorageClassName:              common.LocalStorageName,
			VolumeMode: func() *corev1.PersistentVolumeMode {
				volumeMode := corev1.PersistentVolumeFilesystem
				return &volumeMode
			}(),
			NodeAffinity: &corev1.VolumeNodeAffinity{
				Required: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      common.NodeSelectorKey,
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{common.LokiSelectorVa},
								},
							},
						},
					},
				},
			},
		},
	}

	klog.Infof("start reconcile pv: %s", lokiPv.Name)
	_, err = resources.Reconcile(context.TODO(), k.Client, lokiPv, resources.Option{})
	if err != nil {
		return err
	}

	return nil
}

func makeOverrideIngress(enabled bool, ingressImpl string, host string) map[string]interface{} {
	ing := make(map[string]interface{})
	if enabled {
		ing["enabled"] = enabled
		ing["annotations"] = map[string]interface{}{
			"kubernetes.io/ingress.class": ingressImpl,
		}
		ing["hosts"] = []map[string]interface{}{
			{"host": host, "paths": []string{"/"}},
		}
	} else {
		ing["enabled"] = enabled
	}

	return ing
}

func getIngressName(urlhead string, app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["ing"]; ok {
		return va
	}
	return fmt.Sprintf("%s.loki.dmall.com", urlhead)
}

func getLokiCpuReq(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["loki-req-cpu"]; ok {
		return va
	}
	return "500m"
}

func getLokiCpuLimit(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["loki-limit-cpu"]; ok {
		return va
	}
	return "1"
}

func getLokiMemReq(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["loki-req-mem"]; ok {
		return va
	}
	return "500Mi"
}
func getLokiMemLimit(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["loki-limit-mem"]; ok {
		return va
	}
	return "1Gi"
}

func (r *reconciler) buildLokiStackValues(app *workloadv1beta1.HelmChartSpec) map[string]interface{} {
	err := preInstallLpv(r.k, app, r.obj)
	if err != nil {
		klog.Infof("install pv err: %s", err)
		return nil
	}
	overrideValueMap := map[string]interface{}{
		"loki": map[string]interface{}{
			"enabled": true,
			"ingress": makeOverrideIngress(true, r.ingressImpl, getIngressName(r.urlHead, app)),
			"resources": map[string]interface{}{
				"limits": map[string]interface{}{
					"cpu":    getLokiCpuLimit(app),
					"memory": getLokiMemLimit(app),
				},
				"requests": map[string]interface{}{
					"cpu":    getLokiCpuReq(app),
					"memory": getLokiMemReq(app),
				},
			},
			"persistence": map[string]interface{}{
				"enabled":          true,
				"storageClassName": common.LocalStorageName,
				"accessModes":      []string{"ReadWriteOnce"},
				"size":             getLokiStorageSize(app),
			},
		},
		//"fluent-bit": map[string]interface{}{},
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

	var vaByte []byte
	rlsName, ns, chartURL := common.BuildHelmInfo(app)
	va := r.buildLokiStackValues(app)
	klog.V(4).Infof("rlsName:%s OverrideValue:\n%s", rlsName, va)
	vaByte, err = yaml.Marshal(va)
	if err != nil {
		klog.Errorf("app[%s] Marshal overrideValueMap err:%+v", app.Name, err)
		return nil, err
	}

	klog.V(4).Infof("rlsName:%s OverrideValue:\n%s", rlsName, string(vaByte))
	rls, err := helmv3.ApplyRelease(env, rlsName, chartURL, app.ChartVersion, nil, ns, vaByte, nil)
	if err != nil || rls == nil {
		return nil, err
	}

	return common.ConvertAppHelmReleasePtr(rls), nil
}
