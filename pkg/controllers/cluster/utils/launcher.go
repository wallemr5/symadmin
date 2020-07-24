package utils

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
)

const (
	PodName = "launcher"
)

func makeRootDirMount(pathDirs string) (volume corev1.Volume, mount corev1.VolumeMount) {
	mount = corev1.VolumeMount{
		Name:      "obj-dir",
		MountPath: pathDirs,
	}

	hostType := corev1.HostPathDirectoryOrCreate
	volume = corev1.Volume{
		Name: "obj-dir",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: pathDirs,
				Type: &hostType,
			},
		},
	}
	return
}

func buildLauncherPod(ns string, nodeName string, pathDirs string) *corev1.Pod {
	volume, mount := makeRootDirMount(pathDirs)

	launchArgs := []string{
		"-c",
		fmt.Sprintf("if [ ! -d \"%s\" ]; then\n  mkdir -p %s\nfi", pathDirs, pathDirs),
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", PodName, string(uuid.NewUUID())),
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            PodName,
					Image:           "busybox:1.30",
					Command:         []string{"/bin/sh"},
					Args:            launchArgs,
					VolumeMounts:    []corev1.VolumeMount{mount},
					ImagePullPolicy: corev1.PullIfNotPresent,
				},
			},
			Volumes:       []corev1.Volume{volume},
			NodeName:      nodeName,
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
}

func removePod(cli kubernetes.Interface, pod *corev1.Pod) error {
	err := cli.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "remove pod: %s", pod.Name)
	}
	return nil
}

// ("sym-admin", "10.13.135.252", "/web/1234")
func ApplyLauncherPod(cli kubernetes.Interface, ns string, nodeName string, pathDirs string) error {
	pod := buildLauncherPod(ns, nodeName, pathDirs)
	_, poderr := cli.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	if poderr != nil {
		klog.Errorf("err: %v", poderr)
		return poderr
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		obj, getErr := cli.CoreV1().Pods(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
		if getErr != nil {
			klog.Errorf("get launcher pod err: %v", getErr)
			return getErr
		}

		if obj.Status.Phase != corev1.PodSucceeded {
			klog.Errorf("launcher pod %s is not Succeeded", pod.Name)
		}
		return nil
	})

	if err != nil {
		klog.Errorf("retry err: %v", err)
	} else {
		klog.Infof("apply exec mkdir %s success", pathDirs)
	}
	return removePod(cli, pod)
}
