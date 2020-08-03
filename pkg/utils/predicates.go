package utils

import (
	"reflect"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	pkgLabels "gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// isObserveNamespaces
func isObserveNamespaces(ns string) bool {
	for _, obvNs := range pkgLabels.ObservedNamespace {
		if obvNs == ns {
			return true
		}
	}
	return false
}

// isObserveApp
func isObserveApp(labels map[string]string) bool {
	if _, ok := labels[pkgLabels.ObserveMustLabelAppName]; !ok {
		return false
	}

	if _, ok := labels[pkgLabels.ObserveMustLabelClusterName]; !ok {
		return false
	}
	return true
}

func getObserveApp(labels map[string]string) string {
	if _, ok := labels[pkgLabels.ObserveMustLabelClusterName]; !ok {
		return ""
	}

	if va, ok := labels[pkgLabels.ObserveMustLabelAppName]; ok {
		return va
	}

	return ""
}

func GetWatchPredicateForNs() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return isObserveNamespaces(e.Meta.GetNamespace())
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return isObserveNamespaces(e.Meta.GetNamespace())
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return isObserveNamespaces(e.MetaNew.GetNamespace())
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return isObserveNamespaces(e.Meta.GetNamespace())
		},
	}
}

func GetWatchPredicateForApp() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return isObserveApp(e.Meta.GetLabels())
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return isObserveApp(e.Meta.GetLabels())
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return isObserveApp(e.MetaNew.GetLabels())
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return isObserveApp(e.Meta.GetLabels())
		},
	}
}

func GetEnqueueRequestsMapper() handler.Mapper {
	return handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      getObserveApp(a.Meta.GetLabels()),
					Namespace: a.Meta.GetNamespace(),
				},
			},
		}
	})
}

func GetEnqueueRequestsFucs() handler.EventHandler {
	return handler.Funcs{
		CreateFunc: func(e event.CreateEvent, queue workqueue.RateLimitingInterface) {
			queue.AddRateLimited(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      getObserveApp(e.Meta.GetLabels()),
				Namespace: e.Meta.GetNamespace(),
			}})
		},
		UpdateFunc: func(e event.UpdateEvent, queue workqueue.RateLimitingInterface) {
			queue.AddRateLimited(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      getObserveApp(e.MetaNew.GetLabels()),
				Namespace: e.MetaNew.GetNamespace(),
			}})
		},
		DeleteFunc: func(e event.DeleteEvent, queue workqueue.RateLimitingInterface) {
			queue.AddRateLimited(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      getObserveApp(e.Meta.GetLabels()),
				Namespace: e.Meta.GetNamespace(),
			}})
		},
		GenericFunc: func(e event.GenericEvent, queue workqueue.RateLimitingInterface) {
			queue.AddRateLimited(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      getObserveApp(e.Meta.GetLabels()),
				Namespace: e.Meta.GetNamespace(),
			}})
		},
	}
}

func GetWatchPredicateForAdvDeploymentSpec() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj := e.ObjectOld.(*workloadv1beta1.AdvDeployment)
			newObj := e.ObjectNew.(*workloadv1beta1.AdvDeployment)
			if !equality.Semantic.DeepEqual(oldObj.Spec, newObj.Spec) ||
				IsObjectMetaChange(e.ObjectNew, e.ObjectOld) {
				return true
			}
			return false
		},
	}
}

func GetWatchPredicateForClusterSpec() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj := e.ObjectOld.(*workloadv1beta1.Cluster)
			newObj := e.ObjectNew.(*workloadv1beta1.Cluster)
			if !equality.Semantic.DeepEqual(oldObj.Spec, newObj.Spec) ||
				IsObjectMetaChange(e.ObjectNew, e.ObjectOld) {
				return true
			}
			return false
		},
	}
}

func GetWatchPredicateForAppetSpec() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldObj := e.ObjectOld.(*workloadv1beta1.AppSet)
			newObj := e.ObjectNew.(*workloadv1beta1.AppSet)
			if !equality.Semantic.DeepEqual(oldObj.Spec, newObj.Spec) ||
				IsObjectMetaChange(e.ObjectNew, e.ObjectOld) {
				return true
			}
			return false
		},
	}
}

func isObserveKnownKey(obvKey string) bool {
	for _, key := range pkgLabels.AnnotationsKnownKey {
		if obvKey == key {
			return true
		}
	}
	return false
}

func IsObjectMetaChange(n, c interface{}) bool {
	newObj, ok := n.(metav1.Object)
	if !ok {
		return false
	}

	currObj, ok := c.(metav1.Object)
	if !ok {
		return false
	}

	if !reflect.DeepEqual(newObj.GetFinalizers(), currObj.GetFinalizers()) {
		klog.V(4).Infof("name: %s finalizers changed, new: %+v, curr: %+v",
			newObj.GetName(), newObj.GetFinalizers(), currObj.GetFinalizers())
		return true
	}

	if !reflect.DeepEqual(newObj.GetLabels(), currObj.GetLabels()) {
		klog.V(4).Infof("name: %s labels changed, new: %+v, curr: %+v",
			newObj.GetName(), newObj.GetLabels(), currObj.GetLabels())
		return true
	}

	currAnnotations := map[string]string{}
	for k, v := range currObj.GetAnnotations() {
		if !isObserveKnownKey(k) {
			continue
		}
		currAnnotations[k] = v
	}

	newAnnotations := map[string]string{}
	for k, v := range newObj.GetAnnotations() {
		if !isObserveKnownKey(k) {
			continue
		}
		newAnnotations[k] = v
	}

	if !reflect.DeepEqual(newAnnotations, currAnnotations) {
		klog.V(4).Infof("name: %s annotations changed, new: %+v, curr: %+v",
			newObj.GetName(), newAnnotations, currAnnotations)
		return true
	}

	return false
}

func IsObjectLabelsChange(n, c interface{}) bool {
	newObj, ok := n.(metav1.Object)
	if !ok {
		return false
	}

	currObj, ok := c.(metav1.Object)
	if !ok {
		return false
	}

	if !reflect.DeepEqual(newObj.GetLabels(), currObj.GetLabels()) {
		klog.V(4).Infof("name: %s labels changed, new: %+v, curr: %+v",
			newObj.GetName(), newObj.GetLabels(), currObj.GetLabels())
		return true
	}

	return false
}
