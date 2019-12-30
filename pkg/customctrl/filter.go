package customctrl

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// AnnotationFilterFunc creates a FilterFunc only accepting objects with given annotation key and value
func AnnotationFilterFunc(key string, value string, allowUnset bool) func(interface{}) bool {
	return func(obj interface{}) bool {
		if mo, ok := obj.(metav1.Object); ok {
			anno := mo.GetAnnotations()
			annoVal, ok := anno[key]
			if !ok {
				return allowUnset
			}
			return annoVal == value
		}
		return false
	}
}

// LabelExistsFilterFunc creates a FilterFunc only accepting objects which have a given label.
func LabelExistsFilterFunc(label string) func(obj interface{}) bool {
	return func(obj interface{}) bool {
		if mo, ok := obj.(metav1.Object); ok {
			labels := mo.GetLabels()
			_, ok := labels[label]
			return ok
		}
		return false
	}
}

// LabelFilterFunc creates a FilterFunc only accepting objects where a label is set to a specific value.
func LabelFilterFunc(label string, value string, allowUnset bool) func(interface{}) bool {
	return func(obj interface{}) bool {
		if mo, ok := obj.(metav1.Object); ok {
			labels := mo.GetLabels()
			val, ok := labels[label]
			if !ok {
				return allowUnset
			}
			return val == value
		}
		return false
	}
}

// NameFilterFunc creates a FilterFunc only accepting objects with the given name.
func NameFilterFunc(name string) func(interface{}) bool {
	return func(obj interface{}) bool {
		if mo, ok := obj.(metav1.Object); ok {
			return mo.GetName() == name
		}
		return false
	}
}

// NamespaceFilterFunc creates a FilterFunc only accepting objects in the given namespace.
func NamespaceFilterFunc(namespace string) func(interface{}) bool {
	return func(obj interface{}) bool {
		if mo, ok := obj.(metav1.Object); ok {
			return mo.GetNamespace() == namespace
		}
		return false
	}
}

// ChainFilterFuncs creates a FilterFunc which performs an AND of the passed FilterFuncs.
func ChainFilterFuncs(funcs ...func(interface{}) bool) func(interface{}) bool {
	return func(obj interface{}) bool {
		for _, f := range funcs {
			if !f(obj) {
				return false
			}
		}
		return true
	}
}

// PassNew makes it simple to create an UpdateFunc for use with
// cache.ResourceEventHandlerFuncs that can delegate the same methods
// as AddFunc/DeleteFunc but passing through only the second argument
// (which is the "new" object).
func PassNew(f func(interface{})) func(interface{}, interface{}) {
	return func(first, second interface{}) {
		f(second)
	}
}

// HandlerWraps wraps the provided handler function into a cache.ResourceEventHandler
// that sends all events to the given handler.  For Updates, only the new object
// is forwarded.
func HandlerWraps(h func(interface{})) cache.ResourceEventHandler {
	return cache.ResourceEventHandlerFuncs{
		AddFunc:    h,
		UpdateFunc: PassNew(h),
		DeleteFunc: h,
	}
}

// Filter makes it simple to create FilterFunc's for use with
// cache.FilteringResourceEventHandler that filter based on the
// schema.GroupVersionKind of the controlling resources.
func Filter(gvk schema.GroupVersionKind) func(obj interface{}) bool {
	return func(obj interface{}) bool {
		if object, ok := obj.(metav1.Object); ok {
			owner := metav1.GetControllerOf(object)
			return owner != nil &&
				owner.APIVersion == gvk.GroupVersion().String() &&
				owner.Kind == gvk.Kind
		}
		return false
	}
}
