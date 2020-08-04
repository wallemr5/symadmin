package kube

import (
	"context"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/restmapper"
)

type AnnotationCache struct {
	dynClient dynamic.Interface
	kubeCli   kubernetes.Interface

	cache *lru.ARCCache
	sync.RWMutex
}

func NewAnnotationCache(kubeCli kubernetes.Interface, dynClient dynamic.Interface) *AnnotationCache {
	cache, err := lru.NewARC(1024)
	if err != nil {
		panic("cannot init cache: " + err.Error())
	}
	return &AnnotationCache{
		dynClient: dynClient,
		kubeCli:   kubeCli,
		cache:     cache,
	}
}

func (a *AnnotationCache) GetAnnotationsWithCache(reference *corev1.ObjectReference) (map[string]string, error) {
	uid := reference.UID

	if val, ok := a.cache.Get(uid); ok {
		return val.(map[string]string), nil
	}

	obj, err := GetObject(reference, a.kubeCli, a.dynClient)
	if err == nil {
		annotations := obj.GetAnnotations()
		for key := range annotations {
			if strings.Contains(key, "kubernetes.io/") || strings.Contains(key, "k8s.io/") {
				delete(annotations, key)
			}
		}
		a.cache.Add(uid, annotations)
		return annotations, nil
	}

	if errors.IsNotFound(err) {
		var empty map[string]string
		a.cache.Add(uid, empty)
		return nil, nil
	}

	return nil, err
}

type LabelCache struct {
	dynClient dynamic.Interface
	kubeCli   kubernetes.Interface

	cache *lru.ARCCache
	sync.RWMutex
}

func NewLabelCache(kubeCli kubernetes.Interface, dynClient dynamic.Interface) *LabelCache {
	cache, err := lru.NewARC(1024)
	if err != nil {
		panic("cannot init cache: " + err.Error())
	}
	return &LabelCache{
		dynClient: dynClient,
		kubeCli:   kubeCli,
		cache:     cache,
	}
}

func GetObject(reference *corev1.ObjectReference, kubeCli kubernetes.Interface, dynClient dynamic.Interface) (*unstructured.Unstructured, error) {
	var group, version string
	s := strings.Split(reference.APIVersion, "/")
	if len(s) == 1 {
		group = ""
		version = s[0]
	} else {
		group = s[0]
		version = s[1]
	}

	gk := schema.GroupKind{Group: group, Kind: reference.Kind}

	groupResources, err := restmapper.GetAPIGroupResources(kubeCli.Discovery())
	if err != nil {
		return nil, err
	}

	rm := restmapper.NewDiscoveryRESTMapper(groupResources)
	mapping, err := rm.RESTMapping(gk, version)
	if err != nil {
		return nil, err
	}

	item, err := dynClient.
		Resource(mapping.Resource).
		Namespace(reference.Namespace).
		Get(context.TODO(), reference.Name, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return item, nil
}

func (l *LabelCache) GetLabelsWithCache(reference *corev1.ObjectReference) (map[string]string, error) {
	uid := reference.UID

	if val, ok := l.cache.Get(uid); ok {
		return val.(map[string]string), nil
	}

	obj, err := GetObject(reference, l.kubeCli, l.dynClient)
	if err == nil {
		labels := obj.GetLabels()
		l.cache.Add(uid, labels)
		return labels, nil
	}

	if errors.IsNotFound(err) {
		// There can be events without the involved objects existing, they seem to be not garbage collected?
		// Marking it nil so that we can return faster
		var empty map[string]string
		l.cache.Add(uid, empty)
		return nil, nil
	}

	// An non-ignorable error occurred
	return nil, err
}
