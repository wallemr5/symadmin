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
	"strings"

	"fmt"

	"github.com/ghodss/yaml"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"k8s.io/klog"
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

	klog.V(5).Infof("podSetName:%s overrideValueMap:%v", name, overrideValueMap)
	vaByte, err := yaml.Marshal(overrideValueMap)
	if err != nil {
		klog.Errorf("Marshal overrideValueMap err:%+v", err)
		return ""
	}
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

	if app.Spec.ServiceName != nil {
		lb["lightningDomain0"] = *app.Spec.ServiceName
	}

	lb[labels.LabelKeyZone] = getMapKey(clusterSpec.Mata, labels.LabelKeyZone)
	lb[labels.ObserveMustLabelClusterName] = getMapKey(clusterSpec.Mata, labels.ObserveMustLabelClusterName)

	return lb
}
