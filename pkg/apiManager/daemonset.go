package apiManager

import (
	"context"

	"net/http"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
)

// HandleOfflineWordloadDeploy get log files in a pod
func (m *APIManager) HandleOfflineWordloadDeploy(c *gin.Context) {
	hostPathType := corev1.HostPathDirectory
	lb := map[string]string{
		"app": "offline-pod-log",
	}
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "offline-wordload-ds",
			Namespace: "sym-admin",
			Labels:    lb,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: lb,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: lb,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "web",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/web",
									Type: &hostPathType,
								},
							},
						},
					},
					NodeSelector: map[string]string{
						"beta.kubernetes.io/os": "linux",
					},
					Containers: []corev1.Container{
						{
							Name:  "offline-pod-log",
							Image: "symcn.tencentcloudcr.com/symcn/centos-base:7.8",
							Command: []string{
								"/bin/sh",
								"-c",
								`
set -euo pipefail
while true; do
	time=$(date "+%Y-%m-%d %H:%M:%S")
	echo "Container is Running, now time ${time}"
	sleep 300
done

echo "Container will exit"
`,
							},
							Args: nil,
							Env: []corev1.EnvVar{
								{
									Name:  "NODE_NAME",
									Value: "",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("0.1"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "web",
									ReadOnly:  true,
									MountPath: "/web",
								},
							},
							TerminationMessagePath:   corev1.TerminationMessagePathDefault,
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
							ImagePullPolicy:          corev1.PullIfNotPresent,
						},
					},
					RestartPolicy:                 corev1.RestartPolicyAlways,
					TerminationGracePeriodSeconds: utils.Int64Pointer(5),
					DNSPolicy:                     corev1.DNSClusterFirstWithHostNet,
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoSchedule,
						},
						{
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoExecute,
						},
					},
				},
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: utils.IntstrPointer(1),
				},
			},
		},
	}

	for _, cluster := range m.K8sMgr.GetAll() {
		_, err := resources.Reconcile(context.TODO(), cluster.Client, ds, resources.DesiredStatePresent, true)
		if err != nil {
			klog.Error("cluster: %s apply err: %v", cluster.Name, err)
			AbortHTTPError(c, ParamInvalidError, "", err)
			return
		}
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
	})
}
