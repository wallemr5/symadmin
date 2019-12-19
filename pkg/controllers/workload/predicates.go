package workload

import (
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// isObserveNamespaces
func isObserveNamespaces(ns string) bool {
	for _, obvNs := range utils.ObservedNamespace {
		if obvNs == ns {
			return true
		}
	}
	return false
}

// isObserveApp
func isObserveApp(labels map[string]string) bool {
	if _, ok := labels[utils.ObserveMustLabelAppName]; !ok {
		return false
	}

	if _, ok := labels[utils.ObserveMustLabelClusterName]; !ok {
		return false
	}
	return true
}

func getObserveApp(labels map[string]string) string {
	if _, ok := labels[utils.ObserveMustLabelClusterName]; !ok {
		return ""
	}

	if va, ok := labels[utils.ObserveMustLabelAppName]; ok {
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
