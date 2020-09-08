package v2

import (
	"bytes"
	"errors"

	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog"
)

// RunCmdOnceInContainer ...
func RunCmdOnceInContainer(cluster *k8smanager.Cluster, namespace, pod, container string, cmd []string, tty bool) ([]byte, error) {
	req := cluster.KubeCli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod).
		Namespace(namespace).
		SubResource("exec")

	scheme := runtime.NewScheme()
	if err := core_v1.AddToScheme(scheme); err != nil {
		klog.Errorf("error adding to scheme: %v", err)
		return nil, err
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&core_v1.PodExecOptions{
		Command:   cmd,
		Container: container,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       tty,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(cluster.RestConfig, "POST", req.URL())
	if err != nil {
		klog.Errorf("error while creating Executor: %+v", err)
		return nil, err
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    tty,
	})
	if stderr.Len() > 0 {
		return nil, errors.New(stderr.String())
	}

	if err != nil {
		klog.Errorf("get exec streaming error: %v", err)
		return nil, err
	}

	return stdout.Bytes(), nil
}
