package monitor

import (
	"context"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/common"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster/utils"
	helmv3 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v3"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	pkgLabels "gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type reconciler struct {
	name        string
	mgr         manager.Manager
	k           *k8smanager.Cluster
	obj         *workloadv1beta1.Cluster
	env         *helmv3.HelmEnv
	clusterType string
	urlHead     string
	ingressImpl string
}

// New ...
func New(mgr manager.Manager, k *k8smanager.Cluster, obj *workloadv1beta1.Cluster, env *helmv3.HelmEnv) common.ComponentReconciler {
	r := &reconciler{
		name: "monitor",
		mgr:  mgr,
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

	r.ingressImpl = common.GetIngressImpl(obj.Spec.Meta)
	return r
}

func (r *reconciler) Name() string {
	return r.name
}

func getPromSwitch(clusterType string) (isSysEnable bool, isAlertManagerEnable bool, isGrafanaEnable bool, isIngress bool, isKubeletHTTPS bool) {
	isSysEnable = false
	isAlertManagerEnable = true
	isIngress = true
	isGrafanaEnable = true
	isKubeletHTTPS = true
	if strings.Contains(clusterType, "idc") {
		isSysEnable = true
		return
	}
	if strings.Contains(clusterType, "aks") {
		isKubeletHTTPS = false
		return
	}

	return
}

func getPromStorageSize(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["lpv-size"]; ok {
		return va
	}
	return "30Gi"
}

func getPromPvPath(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["lpv-path"]; ok {
		return va
	}
	return "/root/prometheus-data"
}

func getGrafanaStorageSize(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["lpv-grafana-size"]; ok {
		return va
	}
	return "1Gi"
}

func getGrafanaPvPath(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["lpv-grafana-path"]; ok {
		return va
	}
	return "/root/grafana-data"
}

func getPromLimitsCPU(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["prom-limit-cpu"]; ok {
		return va
	}
	return "1"
}

func getPromLimitsMemory(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["prom-limit-memory"]; ok {
		return va
	}
	return "1Gi"
}

func getPromReqCPU(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["prom-req-cpu"]; ok {
		return va
	}
	return "0.5"
}

func getPromReqMemory(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["prom-req-memory"]; ok {
		return va
	}
	return "500Mi"
}

func getClusterEnv(c *workloadv1beta1.Cluster) string {
	var env string
	if strings.Contains(c.Name, "test") {
		env = "test"
	} else {
		env = "prod"
	}
	return env
}

func getPromSelector(app *workloadv1beta1.HelmChartSpec) map[string]interface{} {
	sel := make(map[string]interface{})

	// default selector all namespace
	if _va, ok := app.Values["selector-only-system"]; ok && _va == "enable" {
		sel["matchExpressions"] = []map[string]interface{}{
			{
				"key":      "name",
				"operator": "In",
				"values":   []string{"default", "kube-system", "monitoring"},
			},
		}
	}

	return sel
}

func getPromRetention(app *workloadv1beta1.HelmChartSpec) string {
	if va, ok := app.Values["prom-retention"]; ok {
		return va
	}
	return "2d"
}

func preInstallMonitoringGetEtcd(k *k8smanager.Cluster) []string {
	nodes := &corev1.NodeList{}
	err := k.Client.List(context.TODO(), nodes, &client.ListOptions{})
	if err != nil {
		klog.Errorf("cluster[%s] list nodes err: %+v", k.Name, err)
		return nil
	}

	nodeIps := []string{}
	for _, node := range nodes.Items {
		if _, ok := node.Labels[common.MasterNodeLabelKey]; ok {
			for _, addr := range node.Status.Addresses {
				if addr.Type == corev1.NodeInternalIP {
					nodeIps = append(nodeIps, addr.Address)
				}
			}
		}
	}

	return nodeIps
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
		return err
	}

	promPath := getPromPvPath(app)
	grafanaPath := getGrafanaPvPath(app)
	if v := pkgLabels.GetAnnotationKey(c.Annotations, pkgLabels.ClusterAnnotationMonitor); v == "" {
		err = utils.ApplyLauncherPod(k.KubeCli, app.Namespace, c.Spec.SymNodeName, promPath)
		if err != nil {
			klog.Errorf("ApplyLauncherPod node: %s path: %s err: %v", c.Spec.SymNodeName, promPath, err)
			return err
		}
		err = utils.ApplyLauncherPod(k.KubeCli, app.Namespace, c.Spec.SymNodeName, grafanaPath)
		if err != nil {
			klog.Errorf("ApplyLauncherPod node: %s path: %s err: %v", c.Spec.SymNodeName, grafanaPath, err)
			return err
		}

		if c.Annotations == nil {
			c.Annotations = make(map[string]string)
		}

		c.Annotations[pkgLabels.ClusterAnnotationMonitor] =
			fmt.Sprintf("{node: %s, prometheusDir: %s, grafanaDir: %s}", c.Spec.SymNodeName, promPath, grafanaPath)
	}

	promPv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   common.PrometheusPvName,
			Labels: pkgLabels.GetLabels(k.Name),
		},
		Spec: corev1.PersistentVolumeSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Capacity: corev1.ResourceList{
				"storage": resource.MustParse(getPromStorageSize(app)),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				Local: &corev1.LocalVolumeSource{
					Path: promPath,
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
									Values:   []string{common.NodeSelectorVa},
								},
							},
						},
					},
				},
			},
		},
	}

	klog.Infof("start reconcile pv: %s", promPv.Name)
	_, err = resources.Reconcile(context.TODO(), k.Client, promPv, resources.Option{})
	if err != nil {
		return err
	}

	grafanaPv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   common.GrafanaPvName,
			Labels: pkgLabels.GetLabels(k.Name),
		},
		Spec: corev1.PersistentVolumeSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Capacity: corev1.ResourceList{
				"storage": resource.MustParse(getGrafanaStorageSize(app)),
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{
				Local: &corev1.LocalVolumeSource{
					Path: grafanaPath,
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
									Values:   []string{common.NodeSelectorVa},
								},
							},
						},
					},
				},
			},
		},
	}

	klog.Infof("start reconcile pv: %s", grafanaPv.Name)
	_, err = resources.Reconcile(context.TODO(), k.Client, grafanaPv, resources.Option{})
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
		ing["hosts"] = []string{host}
	} else {
		ing["enabled"] = enabled
	}

	return ing
}

func getIngressName(urlhead string, componentName string, app *workloadv1beta1.HelmChartSpec) string {
	switch componentName {
	case "grafana":
		if va, ok := app.Values["grafana-ing"]; ok {
			return va
		}
		return fmt.Sprintf("%s.grafana.dmall.com", urlhead)
	case "prometheus":
		if va, ok := app.Values["prom-ing"]; ok {
			return va
		}
		return fmt.Sprintf("%s.prometheus.dmall.com", urlhead)
	case "alertmanager":
		if va, ok := app.Values["alertmanager-ing"]; ok {
			return va
		}
		return fmt.Sprintf("%s.alertmanager.dmall.com", urlhead)
	default:
		return ""
	}
}

func makeAlertManagerConfig(c *workloadv1beta1.Cluster) map[string]interface{} {
	/*
		   global:
		     resolve_timeout: 5m
		   route:
		     group_by: ['job']
		     group_wait: 30s
		     group_interval: 5m
		     repeat_interval: 12h
		     receiver: 'null'
		     routes:
		     - match:
		         alertname: Watchdog
		       receiver: 'null'
		   receivers:
		   - name: 'null'
		---
		    global:
		      resolve_timeout: 5m
		    route:
		      group_by: ['severity','alertname', 'app']
		      group_wait: 30s
		      group_interval: 1m
		      repeat_interval: 5m
		      receiver: 'sym-webhook'
		    receivers:
		    - name: 'sym-webhook'
		      webhook_configs:
		        - url: 'http://api.symphony.dmall.com/operator/promAlert'
	*/
	var ok bool
	var webhookURL string
	ing := make(map[string]interface{})
	if webhookURL, ok = c.Spec.Meta[common.ClusterAlert]; !ok {
		klog.Infof("cluster[%s] not enable alert", c.Name)
		return ing
	}

	if webhookURL == "auto" {
		if strings.Contains(c.Name, "az-hk") {
			webhookURL = "http://api.symphony.inner-dmall.com.hk/operator/promAlert"
		} else {
			webhookURL = "http://api.symphony.dmall.com/operator/promAlert"
		}
	}

	klog.Infof("cluster[%s] alert webhookurl: %s", c.Name, webhookURL)
	ing["global"] = map[string]interface{}{
		"resolve_timeout": "5m",
	}
	ing["route"] = map[string]interface{}{
		"group_by":        []string{"severity", "app", "alertname", "cluster"},
		"group_wait":      "30s",
		"group_interval":  "5m",
		"repeat_interval": "2h",
		"receiver":        "sym-webhook",
		"routes": []map[string]interface{}{
			{
				"receiver": "sym-webhook",
				"match": map[string]interface{}{
					"alertname": "Watchdog",
				},
			},
		},
	}

	ing["receivers"] = []map[string]interface{}{
		{
			"name": "sym-webhook",
			"webhook_configs": []map[string]interface{}{
				{
					"url": webhookURL,
				},
			},
		},
	}
	return ing
}

func (r *reconciler) buildMonitorValues(app *workloadv1beta1.HelmChartSpec) map[string]interface{} {
	var (
		env     string
		etcdips []string
	)

	env = getClusterEnv(r.obj)
	isSysEnable, isAlertManagerEnable, isGrafanaEnable, isIngress, isKubeletHTTPS := getPromSwitch(r.clusterType)

	if isSysEnable {
		etcdips = preInstallMonitoringGetEtcd(r.k)
		klog.Infof("master etcdips: %+v", etcdips)
	}

	err := preInstallLpv(r.k, app, r.obj)
	if err != nil {
		return nil
	}

	affinity := common.MakeNodeAffinity()
	tolerations := common.MakeNodeTolerations()
	clusterName := r.k.Name

	overrideValueMap := map[string]interface{}{
		"prometheus": map[string]interface{}{
			"enabled": true,
			"ingress": makeOverrideIngress(isIngress, r.ingressImpl, getIngressName(r.urlHead, "prometheus", app)),
			"prometheusSpec": map[string]interface{}{
				// "image": map[string]interface{}{
				// 	"repository": RepositoryHub + "prometheus",
				// 	// "tag":        "v2.12.0",
				// },
				"ruleSelectorNilUsesHelmValues":           false,
				"serviceMonitorSelectorNilUsesHelmValues": false,
				"externalLabels": map[string]interface{}{
					"cluster": clusterName,
					"env":     env,
				},
				"replicaExternalLabelNameClear":    true,
				"prometheusExternalLabelNameClear": true,
				"serviceMonitorNamespaceSelector":  getPromSelector(app),
				"ruleNamespaceSelector":            getPromSelector(app),
				"affinity":                         affinity,
				"tolerations":                      tolerations,
				"resources": map[string]interface{}{
					"limits": map[string]interface{}{
						"cpu":    getPromLimitsCPU(app),
						"memory": getPromLimitsMemory(app),
					},
					"requests": map[string]interface{}{
						"cpu":    getPromReqCPU(app),
						"memory": getPromReqMemory(app),
					},
				},
				"retention": getPromRetention(app),
				"storageSpec": map[string]interface{}{
					"volumeClaimTemplate": map[string]interface{}{
						"spec": map[string]interface{}{
							"storageClassName": common.LocalStorageName,
							"accessModes":      []string{"ReadWriteOnce"},
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"storage": getPromStorageSize(app),
								},
							},
						},
					},
				},
				"additionalScrapeConfigs": builAadditionalScrapeConfigs(app),
			},
		},
		"grafana": map[string]interface{}{
			"enabled":       isGrafanaEnable,
			"adminUser":     "admin",
			"adminPassword": "admin",
			"env": map[string]interface{}{
				"GF_AUTH_PROXY_ENABLED":       "true",
				"GF_AUTH_ANONYMOUS_ENABLED":   "true",
				"GF_SECURITY_ALLOW_EMBEDDING": "true",
			},
			"affinity": affinity,
			// "image": map[string]interface{}{
			// 	"tag": "6.4.3",
			// },
			"tolerations": tolerations,
			"persistence": map[string]interface{}{
				"enabled":          true,
				"storageClassName": common.LocalStorageName,
				"size":             getGrafanaStorageSize(app),
			},
			"ingress": makeOverrideIngress(isIngress, r.ingressImpl, getIngressName(r.urlHead, "grafana", app)),
		},
		"alertmanager": map[string]interface{}{
			"enabled": isAlertManagerEnable,
			"config":  makeAlertManagerConfig(r.obj),
			"alertmanagerSpec": map[string]interface{}{
				// "image": map[string]interface{}{
				// 	"repository": RepositoryHub + "alertmanager",
				// 	// "tag":        "v0.17.0",
				// },
				"affinity":    affinity,
				"tolerations": tolerations,
			},
			"ingress": makeOverrideIngress(isIngress, r.ingressImpl, getIngressName(r.urlHead, "alertmanager", app)),
		},
		"kubeApiServer": map[string]interface{}{
			"enabled": true,
		},
		"kubeControllerManager": map[string]interface{}{
			"enabled": isSysEnable,
		},
		"kubeScheduler": map[string]interface{}{
			"enabled": isSysEnable,
		},
		"kubeProxy": map[string]interface{}{
			"enabled": isSysEnable,
		},
		"nodeExporter": map[string]interface{}{
			"enabled": true,
		},
		"kubeStateMetrics": map[string]interface{}{
			"enabled": true,
		},
		"prometheus-node-exporter": map[string]interface{}{
			// "image": map[string]interface{}{
			// 	"repository": RepositoryHub + "node-exporter",
			// },
			"extraArgs": []interface{}{
				"--collector.filesystem.ignored-mount-points=^/(dev|proc|sys|var/lib/docker/.+)($|/)",
				"--collector.filesystem.ignored-fs-types=^(autofs|binfmt_misc|cgroup|configfs|debugfs|devpts|devtmpfs|fusectl|hugetlbfs|mqueue|overlay|proc|procfs|pstore|rpc_pipefs|securityfs|sysfs|tracefs)$",
				"--collector.textfile.directory=/host/prom-textfile",
			},
			"extraHostVolumeMounts": []map[string]interface{}{
				{
					"name":      "prom-textfile",
					"hostPath":  "/opt/prom-textfile",
					"mountPath": "/host/prom-textfile",
				},
			},
		},
		"kube-state-metrics": map[string]interface{}{
			// "image": map[string]interface{}{
			// 	"repository": RepositoryHub + "kube-state-metrics",
			// 	// "tag": "v1.7.2",
			// },
			"affinity":    affinity,
			"tolerations": tolerations,
		},
	}

	if _va, ok := app.Values["custom-resources-config"]; ok && _va == "enable" {
		overrideValueMap["prometheusOperator"] = map[string]interface{}{
			"createCustomResource": false,
			"affinity":             affinity,
			"tolerations":          tolerations,
			"configReloaderCpu":    "500m",
			"configReloaderMemory": "128Mi",
			"resources": map[string]interface{}{
				"limits": map[string]interface{}{
					"cpu":    "500m",
					"memory": "256Mi",
				},
				"requests": map[string]interface{}{
					"cpu":    "200m",
					"memory": "128Mi",
				},
			},
		}
	} else {
		overrideValueMap["prometheusOperator"] = map[string]interface{}{
			"createCustomResource": false,
			"tolerations":          tolerations,
			"affinity":             affinity,
			"admissionWebhooks": map[string]interface{}{
				"patch": map[string]interface{}{
					"tolerations": tolerations,
					"affinity":    affinity,
				},
			},
		}
	}

	overrideValueMap["coreDns"] = map[string]interface{}{
		"enabled": true,
	}
	overrideValueMap["kubeDns"] = map[string]interface{}{
		"enabled": false,
	}

	if isKubeletHTTPS {
		overrideValueMap["kubelet"] = map[string]interface{}{
			"enabled": true,
		}
	} else {
		overrideValueMap["kubelet"] = map[string]interface{}{
			"enabled": true,
			"serviceMonitor": map[string]interface{}{
				"https": false,
			},
		}
	}

	if isSysEnable && len(etcdips) > 0 {
		overrideValueMap["kubeEtcd"] = map[string]interface{}{
			"enabled":   true,
			"endpoints": etcdips,
		}
	} else {
		overrideValueMap["kubeEtcd"] = map[string]interface{}{
			"enabled": false,
		}
	}

	overrideValueMap["defaultRules"] = map[string]interface{}{
		"create": true,
		"rules": map[string]interface{}{
			"alertmanager":  isAlertManagerEnable,
			"kubeScheduler": isSysEnable,
			"etcd":          isSysEnable,
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

	// modify
	if app.ChartName == "" {
		app.ChartName = "prometheus-operator"
	}

	_, ns, chartURL := common.BuildHelmInfo(app)
	// monitor rls name need add cluster name
	rlsName := "monitor-" + r.obj.Name

	va := r.buildMonitorValues(app)
	vaByte, err := yaml.Marshal(va)
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
