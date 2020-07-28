/*
Copyright 2019 The dks authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package appset

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"context"

	"github.com/ghodss/yaml"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/common"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	pkgLabels "gitlab.dmall.com/arch/sym-admin/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func makeHelmOverrideValus(name string, clusterSpec *workloadv1beta1.TargetCluster, app *workloadv1beta1.AppSet) string {
	var podSet *workloadv1beta1.PodSet
	for _, set := range clusterSpec.PodSets {
		if name == set.Name {
			podSet = set
			break
		}
	}

	if podSet == nil {
		return ""
	}

	overrideValueMap := map[string]interface{}{
		"nameOverride":     app.Name,
		"fullnameOverride": name,
		"service":          makeServiceInfo(podSet),
		"sym":              makeSymInfo(podSet, clusterSpec, app),
		"replicaCount":     podSet.Replicas.IntVal,
		"container": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": podSet.Image,
				"tag":        podSet.Version,
			},
			"env":       makeContainerEnv(clusterSpec.Mata, podSet, app),
			"resources": makeResources(podSet),
			"volumeMounts": []map[string]interface{}{
				{
					"name":      "log-path",
					"mountPath": fmt.Sprintf("/web/logs/app/logback/%s", app.Name),
				},
				{
					"name":      "new-log-path",
					"mountPath": fmt.Sprintf("/web/logs/app/aabb/%s", app.Name),
				},
				{
					"name":      "jvm-path",
					"mountPath": fmt.Sprintf("/web/logs/jvm//%s", app.Name),
				},
			},
		},
	}

	// klog.V(5).Infof("podSetName:%s overrideValueMap:%v", name, overrideValueMap)
	vaByte, err := yaml.Marshal(overrideValueMap)
	if err != nil {
		klog.Errorf("Marshal overrideValueMap err:%+v", err)
		return ""
	}
	klog.V(5).Infof("overrideValueMap:%s", string(vaByte))
	return string(vaByte)
}

func makeContainerEnv(clusterMeta map[string]string, podSet *workloadv1beta1.PodSet, app *workloadv1beta1.AppSet) []map[string]interface{} {
	envs := make([]map[string]interface{}, 0)
	if va, ok := clusterMeta[labels.LabelKeyZone]; ok {
		envs = append(envs, map[string]interface{}{
			"name":  "SYM_AVAILABLE_ZONE",
			"value": va,
		})
	}

	if va, ok := clusterMeta[labels.ObserveMustLabelClusterName]; ok {
		envs = append(envs, map[string]interface{}{
			"name":  "SYM_CLUSTER_INFO",
			"value": va,
		})
	}

	if va, ok := podSet.Mata[labels.ObserveMustLabelGroupName]; ok {
		envs = append(envs, map[string]interface{}{
			"name":  "SYM_GROUP",
			"value": va,
		})
	}

	if va, ok := podSet.Mata[labels.ObserveMustLabelLdcName]; ok {
		envs = append(envs, map[string]interface{}{
			"name":  "SYM_LDC",
			"value": va,
		})
	}

	envs = append(envs, map[string]interface{}{
		"name":  "AMP_APP_CODE",
		"value": app.Name,
	})

	envs = append(envs, map[string]interface{}{
		"name":  "AMP_PRO_CODE",
		"value": app.Name,
	})

	envs = append(envs, map[string]interface{}{
		"name":  "AMP_PRO_CODE",
		"value": app.Name,
	})

	envs = append(envs, map[string]interface{}{
		"name":  "SYM_ENABLE_SUBSTITUTE",
		"value": "true",
	})

	envs = append(envs, map[string]interface{}{
		"name":  "MAX_PERM_SIZE",
		"value": "256m",
	})

	envs = append(envs, map[string]interface{}{
		"name":  "RESERVED_SPACE",
		"value": "50m",
	})
	envs = append(envs, map[string]interface{}{
		"name":  "IMAGE_VERSION",
		"value": podSet.Version,
	})
	return envs
}

func makeSymInfo(podSet *workloadv1beta1.PodSet, clusterSpec *workloadv1beta1.TargetCluster, app *workloadv1beta1.AppSet) map[string]interface{} {
	info := map[string]interface{}{}
	info["labels"] = map[string]interface{}{
		labels.ObserveMustLabelGroupName: getMapKey(podSet.Mata, labels.ObserveMustLabelGroupName),
		labels.ObserveMustLabelLdcName:   getMapKey(podSet.Mata, labels.ObserveMustLabelLdcName),
	}

	if app.Spec.ServiceName != nil {
		info["lightningLabels"] = map[string]interface{}{
			"lightningDomain0": *app.Spec.ServiceName,
		}
	}

	info["clusterLabels"] = map[string]interface{}{
		labels.LabelKeyZone:                getMapKey(clusterSpec.Mata, labels.LabelKeyZone),
		labels.ObserveMustLabelClusterName: getMapKey(clusterSpec.Mata, labels.ObserveMustLabelClusterName),
	}
	return info
}

func makeServiceInfo(podSet *workloadv1beta1.PodSet) map[string]interface{} {
	info := map[string]interface{}{}

	enabled := false
	if strings.Contains(podSet.Name, "blue") {
		enabled = true
	}
	info["enabled"] = enabled

	return info
}

func makeResources(podSet *workloadv1beta1.PodSet) map[string]interface{} {
	info := map[string]interface{}{}

	info["limits"] = map[string]interface{}{
		"cpu":    "1",
		"memory": "500Mi",
	}

	info["requests"] = map[string]interface{}{
		"cpu":    "100m",
		"memory": "500Mi",
	}

	return info
}

func getMapKey(target map[string]string, key string) string {
	if va, ok := target[key]; ok {
		return va
	}

	return "null"
}

func makeAdvDeploymentLabel(clusterSpec *workloadv1beta1.TargetCluster, app *workloadv1beta1.AppSet) map[string]string {
	lb := map[string]string{}

	if app.Spec.ServiceName != nil && len(*app.Spec.ServiceName) > 0 {
		lb["lightningDomain0"] = common.FormatToDNS1123(*app.Spec.ServiceName)
	}

	clusterName := getMapKey(clusterSpec.Mata, labels.ObserveMustLabelClusterName)
	if clusterName == "null" {
		clusterName = clusterSpec.Name
	}

	lb[labels.ObserveMustLabelClusterName] = clusterName
	lb[labels.LabelKeyZone] = getMapKey(clusterSpec.Mata, labels.LabelKeyZone)
	return lb
}

func makeAdvDeploymentAnnotation(app *workloadv1beta1.AppSet) map[string]string {
	an := map[string]string{}

	if v, ok := app.Annotations[pkgLabels.WorkLoadAnnotationHpa]; ok {
		an[pkgLabels.WorkLoadAnnotationHpa] = v
	}

	if v, ok := app.Annotations[pkgLabels.WorkLoadAnnotationHpaMetrics]; ok {
		an[pkgLabels.WorkLoadAnnotationHpaMetrics] = v
	}
	return an
}

func mergeVersion(v1, v2 string) string {
	s1 := strings.Split(strings.TrimSpace(v1), "/")
	s2 := strings.Split(strings.TrimSpace(v2), "/")
	m := map[string]struct{}{}
	for _, v := range s1 {
		if v == "" {
			continue
		}
		m[v] = struct{}{}
	}
	for _, v := range s2 {
		if v == "" {
			continue
		}
		m[v] = struct{}{}
	}

	s := make([]int, 0, len(m))
	for k := range m {
		i, _ := strconv.Atoi(strings.TrimLeft(k, "v"))
		s = append(s, i)
	}
	sort.Ints(s)

	r := ""
	for _, k := range s {
		r = fmt.Sprintf("%s/v%d", r, k)
	}

	return strings.Trim(r, "/")
}

// Removes duplicate strings from the slice
func removeDuplicates(slice []*corev1.Event) []*corev1.Event {
	visited := make(map[string]bool, 0)
	result := make([]*corev1.Event, 0)

	for _, elem := range slice {
		if !visited[elem.Reason] {
			visited[elem.Reason] = true
			result = append(result, elem)
		}
	}

	return result
}

func GetAllClustersEventByApp(mgr *k8smanager.ClusterManager, req types.NamespacedName, app *workloadv1beta1.AppSet) ([]*workloadv1beta1.Event, error) {
	events := make([]*corev1.Event, 0)
	evts := make([]*workloadv1beta1.Event, 0)

	eventOption := &client.ListOptions{
		Namespace:     req.Namespace,
		FieldSelector: fields.Set{"type": corev1.EventTypeWarning}.AsSelector(),
	}

	for _, cluster := range app.Spec.ClusterTopology.Clusters {
		c, err := mgr.Get(cluster.Name)
		if err != nil {
			klog.Errorf("cluster[%s] can't find in cluster manager by get event err: %+v", cluster.Name, err)
			continue
		}

		eventList := &corev1.EventList{}
		if err := c.Client.List(context.TODO(), eventList, eventOption); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, fmt.Errorf("cluster[%s] list event fail, err:%+v", c.Name, err)
		}

		for i := range eventList.Items {
			e := &eventList.Items[i]
			if e.InvolvedObject.Kind == "AdvDeployment" && e.InvolvedObject.Name == req.Name {
				events = append(events, e)
				continue
			}

			if e.InvolvedObject.Kind != "Deployment" &&
				e.InvolvedObject.Kind != "StatefulSet" &&
				e.InvolvedObject.Kind != "Pod" {
				continue
			}

			if !labels.CheckEventLabel(e.InvolvedObject.Name, req.Name) {
				continue
			}
			events = append(events, e)
		}
	}

	if len(events) == 0 {
		return evts, nil
	}

	events = removeDuplicates(events)
	for _, event := range events {
		evts = append(evts, &workloadv1beta1.Event{
			Message:         event.Message,
			Reason:          event.Reason,
			Type:            event.Type,
			FirstSeen:       event.FirstTimestamp,
			LastSeen:        event.LastTimestamp,
			Count:           event.Count,
			SourceComponent: event.Source.Component,
			Name:            event.InvolvedObject.Name,
		})
	}

	sort.Slice(evts, func(i int, j int) bool {
		return evts[i].Name > evts[j].Name
	})

	return evts, nil
}

type NameAdvDeployment struct {
	ClusterName string
	Adv         *workloadv1beta1.AdvDeployment
}

func GetAllClustersAdvDeploymentByApp(mgr *k8smanager.ClusterManager, req types.NamespacedName, app *workloadv1beta1.AppSet) ([]*NameAdvDeployment, error) {
	advs := make([]*NameAdvDeployment, 0)

	for _, cluster := range app.Spec.ClusterTopology.Clusters {
		c, err := mgr.Get(cluster.Name)
		if err != nil {
			klog.Errorf("cluster[%s] can't find in cluster manager by get AdvDeployment err: %+v", cluster.Name, err)
			continue
		}

		obj := &workloadv1beta1.AdvDeployment{}
		if err = c.Client.Get(context.TODO(), req, obj); err != nil {
			if apierrors.IsNotFound(err) {
				klog.Warningf("cluster[%s] can't find AdvDeployment ???", cluster.Name)
				continue
			}
			return nil, fmt.Errorf("cluster[%s] get AdvDeployment fail, err:%+v", cluster.Name, err)
		}

		advs = append(advs, &NameAdvDeployment{
			ClusterName: cluster.Name,
			Adv:         obj,
		})
	}

	return advs, nil
}
