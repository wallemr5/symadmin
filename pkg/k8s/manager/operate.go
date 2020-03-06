package manager

import (
	"context"
	"errors"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	rlsv2 "k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BaseCluster is used to shield the complexity of the underlying multi-cluster and single cluster.
type BaseCluster interface {
	GetOriginKubeCli(clusterNames ...string) (kubernetes.Interface, error)
	GetRestConfig(clusterNames ...string) (*rest.Config, error)
	GetPod(opts types.NamespacedName, clusterNames ...string) (*corev1.Pod, error)
	GetPods(opts *client.ListOptions, clusterNames ...string) ([]*corev1.Pod, error)
	GetNodes(opts *client.ListOptions, clusterNames ...string) ([]*corev1.Node, error)
	GetDeployment(opts *client.ListOptions, clusterNames ...string) ([]*appv1.Deployment, error)
	GetService(opts *client.ListOptions, clusterNames ...string) ([]*corev1.Service, error)
	GetEndpoints(opts *client.ListOptions, clusterNames ...string) ([]*corev1.Endpoints, error)
	GetEvent(opts *client.ListOptions, clusterNames ...string) ([]*corev1.Event, error)
	DeletePods(opts *client.ListOptions, clusterNames ...string) error
	DeletePod(opts types.NamespacedName, clusterNames ...string) error
}

// CustomeCluster extend the methods of the basic cluster, including some special business methods.
type CustomeCluster interface {
	BaseCluster
	GetHelmRelease(opts map[string]string, clusterNames ...string) ([]*rlsv2.Release, error)
}

// GetOriginKubeCli returns the kubecli of the master cluster client if len(clusterNames) == 0, otherwise
// returns the specific kubecli.
func (m *ClusterManager) GetOriginKubeCli(clusterNames ...string) (kubernetes.Interface, error) {
	if len(clusterNames) > 0 {
		if len(clusterNames) > 1 {
			return nil, errors.New("too many clusterNames")
		}
		cluster, err := m.Get(clusterNames[0])
		if err != nil {
			return nil, err
		}
		return cluster.KubeCli, nil
	}
	return m.KubeCli, nil
}

// GetRestConfig returns the rest.Config of the master cluster client if len(clusterNames) == 0, otherwise
// returns the specific rest.Config.
func (m *ClusterManager) GetRestConfig(clusterNames ...string) (*rest.Config, error) {
	if len(clusterNames) > 0 {
		if len(clusterNames) > 1 {
			return nil, errors.New("too many clusterNames")
		}
		cluster, err := m.Get(clusterNames[0])
		if err != nil {
			return nil, err
		}
		return cluster.RestConfig, nil
	}
	return m.Manager.GetConfig(), nil
}

// GetPod ...
func (m *ClusterManager) GetPod(opts types.NamespacedName, clusterNames ...string) (*corev1.Pod, error) {
	clusters := m.GetAll(clusterNames...)
	ctx := context.Background()
	pod := &corev1.Pod{}

	for _, cluster := range clusters {
		err := cluster.Client.Get(ctx, opts, pod)
		if err != nil {
			return nil, err
		}
	}
	return pod, nil
}

// GetPods ...
func (m *ClusterManager) GetPods(opts *client.ListOptions, clusterNames ...string) ([]*corev1.Pod, error) {
	clusters := m.GetAll(clusterNames...)
	ctx := context.Background()
	result := make([]*corev1.Pod, 4)

	for _, cluster := range clusters {
		podList := &corev1.PodList{}
		err := cluster.Client.List(ctx, opts, podList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		for _, pod := range podList.Items {
			result = append(result, &pod)
		}
	}
	return result, nil
}

// GetNodes ...
func (m *ClusterManager) GetNodes(opts *client.ListOptions, clusterNames ...string) ([]*corev1.Node, error) {
	clusters := m.GetAll(clusterNames...)
	ctx := context.Background()
	result := make([]*corev1.Node, 4)

	for _, cluster := range clusters {
		nodeList := &corev1.NodeList{}
		err := cluster.Client.List(ctx, opts, nodeList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		for _, node := range nodeList.Items {
			result = append(result, &node)
		}
	}
	return result, nil
}

// GetDeployment ...
func (m *ClusterManager) GetDeployment(opts *client.ListOptions, clusterNames ...string) ([]*appv1.Deployment, error) {
	clusters := m.GetAll(clusterNames...)
	ctx := context.Background()
	result := make([]*appv1.Deployment, 4)

	for _, cluster := range clusters {
		deployList := &appv1.DeploymentList{}
		err := cluster.Client.List(ctx, opts, deployList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		for _, deploy := range deployList.Items {
			result = append(result, &deploy)
		}
	}
	return result, nil
}

// GetService ...
func (m *ClusterManager) GetService(opts *client.ListOptions, clusterNames ...string) ([]*corev1.Service, error) {
	clusters := m.GetAll(clusterNames...)
	ctx := context.Background()
	result := make([]*corev1.Service, 4)

	for _, cluster := range clusters {
		serviceList := &corev1.ServiceList{}
		err := cluster.Client.List(ctx, opts, serviceList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		for _, service := range serviceList.Items {
			result = append(result, &service)
		}
	}
	return result, nil
}

// GetEndpoints ...
func (m *ClusterManager) GetEndpoints(opts *client.ListOptions, clusterNames ...string) ([]*corev1.Endpoints, error) {
	clusters := m.GetAll(clusterNames...)
	ctx := context.Background()
	result := make([]*corev1.Endpoints, 4)

	for _, cluster := range clusters {
		endpointsList := &corev1.EndpointsList{}
		err := cluster.Client.List(ctx, opts, endpointsList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		for _, endpoints := range endpointsList.Items {
			result = append(result, &endpoints)
		}
	}
	return result, nil
}

// GetEvent ...
func (m *ClusterManager) GetEvent(opts *client.ListOptions, clusterNames ...string) ([]*corev1.Event, error) {
	clusters := m.GetAll(clusterNames...)
	ctx := context.Background()
	result := make([]*corev1.Event, 4)

	for _, cluster := range clusters {
		eventList := &corev1.EventList{}
		err := cluster.Client.List(ctx, opts, eventList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return nil, err
		}
		for _, event := range eventList.Items {
			result = append(result, &event)
		}
	}
	return result, nil
}

// DeletePods ...
func (m *ClusterManager) DeletePods(opts *client.ListOptions, clusterNames ...string) error {
	clusters := m.GetAll(clusterNames...)
	ctx := context.Background()

	for _, cluster := range clusters {
		podList := &corev1.PodList{}
		err := cluster.Client.List(ctx, opts, podList)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return err
		}
		for _, pod := range podList.Items {
			err = cluster.Client.Delete(ctx, &pod)
			if err != nil {
				klog.Errorf("delete pod error: %v", err)
			}
		}
	}
	return nil
}

// DeletePod ...
func (m *ClusterManager) DeletePod(opts types.NamespacedName, clusterNames ...string) error {
	clusters := m.GetAll(clusterNames...)
	ctx := context.Background()
	pod := &corev1.Pod{}

	for _, cluster := range clusters {
		err := cluster.Client.Get(ctx, opts, pod)
		if err != nil {
			return err
		}
		err = cluster.Client.Delete(ctx, pod)
		if err != nil {
			klog.Errorf("delete pod error: %v", err)
		}
	}
	return nil
}
