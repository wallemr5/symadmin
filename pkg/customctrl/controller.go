package customctrl

import (
	"context"
	"fmt"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sync"
	"time"
)

var (
	// DefaultThreadsPerController is the number of threads to use when processing the controller's workqueue.
	DefaultThreadsPerController = 2

	// DefaultMaxRetries
	DefaultMaxRetries = 10
)

type CustomRequest struct {
	reconcile.Request

	ClusterName string
}

// Reconciler is the interface that controller implementations are expected to implement
type CustomReconciler interface {
	CustomReconcile(ctx context.Context, req CustomRequest) (ctrl.Result, error)
}

// Impl is our core controller implementation.  It handles queuing and feeding work
// from the queue to an implementation of Reconciler.
type Impl struct {
	// Name is controller name and workqueue name
	Name string

	Namespaces []string

	MaxRetries  int
	Threadiness int

	// Started is true if the Controller has been Started
	Started bool

	// mu is used to synchronize Controller setup
	mu sync.Mutex

	// Reconciler is the workhorse of this controller
	Reconciler CustomReconciler

	// WorkQueue is a rate limited work queue.
	WorkQueue workqueue.RateLimitingInterface
}

// NewImpl instantiates an instance of our controller
func NewImpl(r CustomReconciler, name string, maxRetries, threadiness *int, namespaces ...string) *Impl {
	impl := &Impl{
		Name:       name,
		Namespaces: namespaces,
		Reconciler: r,
		WorkQueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), name),
	}

	if maxRetries != nil {
		impl.MaxRetries = *maxRetries
	} else {
		impl.MaxRetries = DefaultMaxRetries
	}

	if threadiness != nil {
		impl.Threadiness = *threadiness
	} else {
		impl.Threadiness = DefaultThreadsPerController
	}
	return impl
}

// EnqueueAfter takes a resource, converts it into a namespace/name string, and passes it to EnqueueKey.
func (c *Impl) EnqueueAfter(obj interface{}, after time.Duration) {
	object, err := DeletionHandlingAccessor(obj)
	if err != nil {
		klog.Errorf("DeletionHandlingAccessor err: %#v", err)
		return
	}
	c.EnqueueKeyAfter(object.GetNamespace(), object.GetName(), after)
}

// Enqueue takes a resource, converts it into a namespace/name string, and passes it to EnqueueKey.
func (c *Impl) Enqueue(obj interface{}) {
	object, err := DeletionHandlingAccessor(obj)
	if err != nil {
		klog.Errorf("DeletionHandlingAccessor err: %#v", err)
		return
	}
	c.EnqueueKey(object.GetNamespace(), object.GetName())
}

// get ClusterName from obj label
func getClusterByLabels(obj interface{}) string {
	object, err := DeletionHandlingAccessor(obj)
	if err != nil {
		return ""
	}

	if name, ok := object.GetLabels()[labels.LabelClusterName]; ok {
		return name
	}

	if name, ok := object.GetLabels()[labels.ObserveMustLabelClusterName]; ok {
		return name
	}

	return ""
}

// Enqueue takes a resource, converts it into a clusterName/namespace/name string, and passes it to EnqueueKey.
func (c *Impl) EnqueueMulti(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Errorf("Enqueue err: %#v", err)
		return
	}
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		klog.Errorf("SplitMetaNamespaceKey key:%s err: %#v", key, err)
		return
	}

	c.EnqueueKeyRateLimited(ns, name, getClusterByLabels(obj))
}

func (c *Impl) EnqueueMultiAfter(obj interface{}, after time.Duration) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		klog.Errorf("Enqueue err: %#v", err)
		return
	}

	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		klog.Errorf("SplitMetaNamespaceKey key:%s err: %#v", key, err)
		return
	}
	c.EnqueueKeyAfter(ns, name, after, getClusterByLabels(obj))
}

// Enqueue takes a resource, converts it into a clusterName/namespace/name string, and passes it to EnqueueKey.
func (c *Impl) EnqueueMultiLabelOfCluster(obj interface{}) {
	object, err := DeletionHandlingAccessor(obj)
	if err != nil {
		klog.Errorf("err: %#v", err)
		return
	}

	lb := object.GetLabels()
	if _, ok := lb[labels.ObserveMustLabelLdcName]; !ok {
		klog.V(4).Infof("Object %s/%s does not have a referring label: %s",
			object.GetNamespace(), object.GetName(), labels.ObserveMustLabelLdcName)
		return
	}

	if _, ok := lb[labels.ObserveMustLabelClusterName]; !ok {
		klog.V(4).Infof("Object %s/%s does not have a referring label: %s",
			object.GetNamespace(), object.GetName(), labels.ObserveMustLabelClusterName)
		return
	}

	controllerKey, ok := lb[labels.ObserveMustLabelAppName]
	if !ok {
		klog.V(4).Infof("Object %s/%s does not have a referring label: %s",
			object.GetNamespace(), object.GetName(), labels.ObserveMustLabelAppName)
		return
	}

	klog.V(4).Infof("enqueue ns:%s, name:%s by name:%s", object.GetNamespace(), controllerKey, object.GetName())
	c.EnqueueKeyRateLimited(object.GetNamespace(), controllerKey, getClusterByLabels(obj))
}

// EnqueueControllerOf takes a resource, identifies its controller resource,
// converts it into a namespace/name string, and passes that to EnqueueKey.
func (c *Impl) EnqueueControllerOf(obj interface{}) {
	object, err := DeletionHandlingAccessor(obj)
	if err != nil {
		klog.Errorf("Enqueue err: %#v", err)
		return
	}

	// If we can determine the controller ref of this object, then
	// add that object to our workqueue.
	if owner := metav1.GetControllerOf(object); owner != nil {
		c.EnqueueKey(object.GetNamespace(), owner.Name)
	}
}

// EnqueueLabelOfNamespaceScopedResource returns with an Enqueue func that
// takes a resource, identifies its controller resource through given namespace
// and name labels, converts it into a namespace/name string, and passes that
// to EnqueueKey. The controller resource must be of namespace-scoped.
func (c *Impl) EnqueueLabelOfNamespaceScopedResource(namespaceLabel, nameLabel string) func(obj interface{}) {
	return func(obj interface{}) {
		object, err := DeletionHandlingAccessor(obj)
		if err != nil {
			klog.Errorf("err: %#v", err)
			return
		}

		lb := object.GetLabels()
		controllerKey, ok := lb[nameLabel]
		if !ok {
			klog.Infof("Object %s/%s does not have a referring name label %s",
				object.GetNamespace(), object.GetName(), nameLabel)
			return
		}

		if namespaceLabel != "" {
			controllerNamespace, ok := lb[namespaceLabel]
			if !ok {
				klog.Infof("Object %s/%s does not have a referring namespace label %s",
					object.GetNamespace(), object.GetName(), namespaceLabel)
				return
			}

			c.EnqueueKey(controllerNamespace, controllerKey)
			return
		}

		// Pass through namespace of the object itself if no namespace label specified.
		// This is for the scenario that object and the parent resource are of same namespace,
		// e.g. to enqueue the revision of an endpoint.
		c.EnqueueKey(object.GetNamespace(), controllerKey)
	}
}

// EnqueueLabelOfClusterScopedResource returns with an Enqueue func
// that takes a resource, identifies its controller resource through
// given name label, and passes it to EnqueueKey.
// The controller resource must be of cluster-scoped.
func (c *Impl) EnqueueLabelOfClusterScopedResource(nameLabel string) func(obj interface{}) {
	return func(obj interface{}) {
		object, err := DeletionHandlingAccessor(obj)
		if err != nil {
			klog.Errorf("err: %#v", err)
			return
		}

		lb := object.GetLabels()
		controllerKey, ok := lb[nameLabel]
		if !ok {
			klog.Infof("Object %s/%s does not have a referring name label %s",
				object.GetNamespace(), object.GetName(), nameLabel)
			return
		}

		c.EnqueueKey(object.GetNamespace(), controllerKey)
	}
}

// isObserveNamespaces
func (c *Impl) isObserveNamespaces(ns string) bool {
	if len(c.Namespaces) < 1 {
		return true
	}

	for _, obNs := range c.Namespaces {
		if obNs == ns {
			return true
		}
	}
	return false
}

// EnqueueKey takes a clusterName/namespace/name string and puts it onto the work queue.
func (c *Impl) EnqueueKey(namespace, name string, clusterName ...string) {
	req := c.GetCustomRequest(namespace, name, clusterName...)
	if req == nil {
		return
	}

	c.WorkQueue.Add(*req)
}

// EnqueueKey takes a clusterName/namespace/name string and puts it onto the work queue.
func (c *Impl) EnqueueKeyRateLimited(namespace, name string, clusterName ...string) {
	req := c.GetCustomRequest(namespace, name, clusterName...)
	if req == nil {
		return
	}

	c.WorkQueue.AddRateLimited(*req)
}

func (c *Impl) GetCustomRequest(namespace, name string, clusterName ...string) *CustomRequest {
	if !c.isObserveNamespaces(namespace) {
		return nil
	}

	req := &CustomRequest{
		Request: reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			},
		},
	}

	if len(clusterName) > 0 && clusterName[0] != "" {
		req.ClusterName = utils.ToClusterCrName(clusterName[0])
	}

	return req
}

// EnqueueKeyAfter takes a clusterName/namespace/name string and schedules its execution in the work queue after given delay.
func (c *Impl) EnqueueKeyAfter(namespace, name string, delay time.Duration, clusterName ...string) {
	req := c.GetCustomRequest(namespace, name, clusterName...)
	if req == nil {
		return
	}

	c.WorkQueue.AddAfter(*req, delay)
}

// Run starts the controller's worker threads, the number of which is threadiness.
// It then blocks until stopCh is closed, at which point it shuts down its internal
// work queue and waits for workers to finish processing their current work items.
func (c *Impl) Start(stopCh <-chan struct{}) error {
	c.mu.Lock()

	defer runtime.HandleCrash()
	defer c.WorkQueue.ShutDown()

	// Launch workers to process resources that get enqueued to our workqueue.
	klog.Infof("Starting workers name: %s threadiness: %d", c.Name, c.Threadiness)
	for i := 0; i < c.Threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	c.Started = true
	c.mu.Unlock()

	<-stopCh
	klog.Infof("Shutting down workers, name: %s", c.Name)
	return nil
}

// runWorker is loop warp
func (c *Impl) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling Reconcile on our Reconciler.
func (c *Impl) processNextWorkItem() bool {
	obj, shutdown := c.WorkQueue.Get()
	if shutdown {
		// Stop working
		return false
	}

	startTime := time.Now()
	defer c.WorkQueue.Done(obj)

	var req CustomRequest
	var ok bool

	defer func() {
		// todo Update metrics after processing each item

		diffTime := time.Since(startTime)
		var logLevel klog.Level
		if diffTime > 2*time.Second {
			logLevel = 2
		} else if diffTime > 1*time.Second {
			logLevel = 3
		} else {
			logLevel = 4
		}
		klog.V(logLevel).Infof("Name:%s Reconcile succeeded. Time taken: %v. key: %v", c.Name, diffTime, req)
	}()

	if req, ok = obj.(CustomRequest); !ok {
		c.WorkQueue.Forget(obj)
		runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		return true
	}

	// Run Reconcile, passing it the namespaces/namespace/name string of the resource to be synced.
	if res, err := c.Reconciler.CustomReconcile(context.TODO(), req); err != nil {
		// c.handleErr(err, key)
		c.WorkQueue.AddRateLimited(req)
		klog.V(3).Infof("Name: %s Reconcile failed. Time taken: %v. req: %v", c.Name, time.Since(startTime), req)
		return true
	} else if res.RequeueAfter > 0 {
		// The result.RequeueAfter request will be lost, if it is returned
		// along with a non-nil error. But this is intended as
		// We need to drive to stable reconcile loops before queuing due
		// to result.RequestAfter
		c.WorkQueue.Forget(obj)
		c.WorkQueue.AddAfter(req, res.RequeueAfter)
		return true
	} else if res.Requeue {
		c.WorkQueue.AddRateLimited(req)
		return true
	}

	// Finally, if no error occurs we Forget this item so it does not have any delay when another change happens.
	c.WorkQueue.Forget(req)
	return true
}

func (c *Impl) handleErr(err error, key string) {
	// Re-queue the key if it's an transient error.
	if !IsPermanentError(err) {
		// This controller retries 5 times if something goes wrong. After that, it stops trying.
		if c.WorkQueue.NumRequeues(key) < c.MaxRetries {
			klog.Infof("Error syncing key %v: %v", key, err)

			// Re-enqueue the key rate limited. Based on the rate limiter on the
			// queue and the re-enqueue history, the key will be processed later again.
			c.WorkQueue.AddRateLimited(key)
			return
		}
	}

	klog.Errorf("Reconcile error: %#v, key: %s", err, key)
	c.WorkQueue.Forget(key)
}

// GlobalResync enqueues all objects from the passed SharedInformer
func (c *Impl) GlobalResync(si cache.SharedInformer) {
	for _, rawReq := range si.GetStore().List() {
		req := rawReq.(CustomRequest)
		c.WorkQueue.AddRateLimited(req)
	}
}

// NewPermanentError returns a new instance of permanentError.
// Users can wrap an error as permanentError with this in reconcile,
// when he does not expect the key to get re-queued.
func NewPermanentError(err error) error {
	return permanentError{e: err}
}

// permanentError is an error that is considered not transient.
// We should not re-queue keys when it returns with thus error in reconcile.
type permanentError struct {
	e error
}

// IsPermanentError returns true if given error is permanentError
func IsPermanentError(err error) bool {
	switch err.(type) {
	case permanentError:
		return true
	default:
		return false
	}
}

// Error implements the Error() interface of error.
func (err permanentError) Error() string {
	if err.e == nil {
		return ""
	}

	return err.e.Error()
}

// Informer is the group of methods that a type must implement to be passed to
// StartInformers.
type Informer interface {
	Run(<-chan struct{})
	HasSynced() bool
}

// StartInformers kicks off all of the passed informers and then waits for all
// of them to synchronize.
func StartInformers(stopCh <-chan struct{}, informers ...Informer) error {
	for _, informer := range informers {
		informer := informer
		go informer.Run(stopCh)
	}

	for i, informer := range informers {
		if ok := cache.WaitForCacheSync(stopCh, informer.HasSynced); !ok {
			return fmt.Errorf("Failed to wait for cache at index %d to sync", i)
		}
	}
	return nil
}

// StartAll kicks off all of the passed controllers.
func StartAll(stopCh <-chan struct{}, controllers ...*Impl) {
	wg := sync.WaitGroup{}
	// Start all of the controllers.
	for _, ctrlr := range controllers {
		wg.Add(1)
		go func(c *Impl) {
			defer wg.Done()
			_ = c.Start(stopCh)
		}(ctrlr)
	}
	wg.Wait()
}
