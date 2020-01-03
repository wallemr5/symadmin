package appset

import (
	"strings"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	"gopkg.in/yaml.v2"
)

func MockPodRawValues(info *workloadv1beta1.AppSet) bool {
	changeStatus := false

	for i := 0; i < len(info.Spec.ClusterTopology.Clusters); i++ {
		replica := 0
		for _, v := range info.Spec.ClusterTopology.Clusters[i].PodSets {
			replica += v.Replicas.IntValue()
		}

		for j := 0; j < len(info.Spec.ClusterTopology.Clusters[i].PodSets); j++ {
			if info.Spec.ClusterTopology.Clusters[i].PodSets[j].RawValues != "" {
				continue
			}
			changeStatus = true

			envs := []Env{}
			for k, v := range info.Spec.ClusterTopology.Clusters[i].Mata {
				envs = append(envs, Env{
					Name:  strings.ToUpper(k),
					Value: v,
				})
			}

			rawValues := RawValues{
				NameOverride:     info.Name,
				FullnameOverride: info.Spec.ClusterTopology.Clusters[i].PodSets[j].Name,
				Service: Service{
					Enabled: info.Spec.ClusterTopology.Clusters[i].PodSets[j].Mata[utils.ObserveMustLabelGroupName] == "blue",
				},
				Sym: Sym{
					Env:    envs,
					Labels: info.Spec.ClusterTopology.Clusters[i].PodSets[j].Mata,
					LightningLabels: map[string]string{
						"lightningDomain0": *info.Spec.ServiceName,
					},
					ClusterLabels: info.Spec.ClusterTopology.Clusters[i].PodSets[j].Mata,
				},
				ReplicaCount: replica,
				Container: Container{
					Image: Image{
						Repository: info.Spec.ClusterTopology.Clusters[i].PodSets[j].Image,
						Tag:        info.Spec.ClusterTopology.Clusters[i].PodSets[j].Version,
					},
					Env: []Env{
						Env{Name: "SYM_ENABLE_SUBSTITUTE", Value: "true"},
						Env{Name: "SYM_GROUP", Value: info.Spec.ClusterTopology.Clusters[i].PodSets[j].Mata[utils.ObserveMustLabelGroupName]},
						Env{Name: "AMP_APP_CODE", Value: info.Name},
						Env{Name: "AMP_PRO_CODE", Value: info.Name},
						Env{Name: "APP_CODE", Value: info.Name},
						Env{Name: "MAX_PERM_SIZE", Value: "256m"},
						Env{Name: "RESERVED_SPACE", Value: "50m"},
					},
					Resources: map[string]map[string]string{
						"limits": map[string]string{
							"cpu":    "1",
							"memory": "500Mi",
						},
						"requests": map[string]string{
							"cpu":    "100m",
							"memory": "500Mi",
						},
					},
					VolumeMounts: []VolumeMount{
						VolumeMount{MountPath: "/web/logs/app/logback/" + info.Name, Name: "log-path"},
						VolumeMount{MountPath: "/web/logs/app/aabb/" + info.Name, Name: "new-log-path"},
						VolumeMount{MountPath: "/web/logs/jvm/", Name: "jvm-path"},
					},
				},
			}

			b, _ := yaml.Marshal(rawValues)
			info.Spec.ClusterTopology.Clusters[i].PodSets[j].RawValues = string(b)
		}
	}
	return changeStatus
}

type RawValues struct {
	NameOverride     string    `yaml:"nameOverride"`
	FullnameOverride string    `yaml:"fullnameOverride"`
	Service          Service   `yaml:"service"`
	Sym              Sym       `yaml:"sym"`
	ReplicaCount     int       `yaml:"replicaCount"`
	Container        Container `yaml:"container"`
}

type Service struct {
	Enabled bool `yaml:"enabled"`
}

type Sym struct {
	Env             []Env             `yaml:"env"`
	Labels          map[string]string `yaml:"labels"`
	LightningLabels map[string]string `yaml:"lightningLabels"`
	ClusterLabels   map[string]string `yaml:"clusterLabels"`
}

type Env struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Container struct {
	Image        Image                        `yaml:"image"`
	Env          []Env                        `yaml:"env"`
	Resources    map[string]map[string]string `yaml:"resources"`
	VolumeMounts []VolumeMount                `yaml:"volumeMounts"`
}

type Image struct {
	Repository string `yaml:"repository"`
	Tag        string `yaml:"tag"`
}

type VolumeMount struct {
	MountPath string `yaml:"mountPath"`
	Name      string `yaml:"name"`
}

// nameOverride: "bbcc"
// fullnameOverride: "bbcc-gz01b-blue"

// service:
//   enabled: true

// sym:
//   env:
//     - name: SYM_AVAILABLE_ZONE
//       value: BJ5
//     - name: SYM_CLUSTER_INFO
//       value: tcc-bj5-dks-monit-01
//   labels:
//     sym-group: blue
//     sym-ldc: gz01b
//   lightningLabels:
//     lightningDomain0: outer.bbcc.dmall.com
//   clusterLabels:
//     sym-available-zone: bj5
//     sym-cluster-info: tcc-bj5-dks-monit-01

// replicaCount: 2
// container:
//   image:
//     repository: registry.cn-hangzhou.aliyuncs.com/dmall/bbcc
//     tag: v1
//   env:
//     - name: SYM_ENABLE_SUBSTITUTE
//       value: 'true'
//     - name: SYM_GROUP
//       value: blue
//     - name: AMP_APP_CODE
//       value: bbcc
//     - name: AMP_PRO_CODE
//       value: bbcc
//     - name: APP_CODE
//       value: bbcc
//     - name: MAX_PERM_SIZE
//       value: 256m
//     - name: RESERVED_SPACE
//       value: 50m
//   resources:
//     limits:
//       cpu: "1"
//       memory: 500Mi
//     requests:
//       cpu: 100m
//       memory: 500Mi
//   volumeMounts:
//     - mountPath: /web/logs/app/logback/bbcc
//       name: log-path
//     - mountPath: /web/logs/app/aabb/bbcc
//       name: new-log-path
//     - mountPath: /web/logs/jvm/bbcc
//       name: jvm-path
