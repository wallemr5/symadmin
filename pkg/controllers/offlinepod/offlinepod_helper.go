package offlinepod

import (
	"context"
	"time"

	"sync"

	"github.com/go-logr/logr"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	controllerName = "offlinepod-controller"
	ConfigDataKey  = "offlineList"
)

type offlinepodImpl struct {
	Name       string
	Namespaces []string
	WorkQueue  workqueue.RateLimitingInterface
	MasterMgr  manager.Manager
	client.Client
	Log            logr.Logger
	Cache          map[string]*Cache
	OfflinePodPool *sync.Pool
	sync.RWMutex
	MaxOffline int32
}

func getAppName(lb map[string]string) string {
	name, ok := lb[labels.ObserveMustLabelAppName]
	if !ok {
		return ""
	}

	return name
}

func NewOfflinepodReconciler(mgr manager.Manager, cMgr *pkgmanager.DksManager) (*offlinepodImpl, error) {
	impl := &offlinepodImpl{
		Name:       controllerName,
		Namespaces: labels.ObservedNamespace,
		WorkQueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), controllerName),
		MasterMgr:  mgr,
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName(controllerName),
		Cache:      make(map[string]*Cache),
		OfflinePodPool: &sync.Pool{
			New: func() interface{} {
				return &model.OfflinePod{}
			}},
		MaxOffline: 2,
	}

	for _, cluster := range cMgr.K8sMgr.GetAll() {
		podInformer, err := cluster.Cache.GetInformer(&corev1.Pod{})
		if err != nil {
			klog.Errorf("cluster name:%s can't add pod InformerEntry, err: %v", cluster.Name, err)
			continue
		}

		clusterName := cluster.Name
		podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				var ok bool
				if _, ok = obj.(metav1.Object); !ok {
					// If the object doesn't have Metadata, assume it is a tombstone object of type DeletedFinalStateUnknown
					tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
					if !ok {
						klog.Errorf("Error decoding objects.  Expected cache.DeletedFinalStateUnknown, type: %T", obj)
						return
					}

					// Set obj to the tombstone obj
					obj = tombstone.Obj
					klog.Infof("delete key: %s", tombstone.Key)
				}

				pod, ok := obj.(*corev1.Pod)
				if !ok {
					klog.Errorf("OnDelete missing runtime.Object, type: %T", obj)
					return
				}

				if !impl.isObserveNamespaces(pod.Namespace) {
					return
				}

				if len(pod.Status.HostIP) == 0 {
					return
				}

				oPod := impl.GetOfflinePod()
				oPod.Name = pod.Name
				oPod.ClusterName = clusterName
				oPod.AppName = getAppName(pod.Labels)
				oPod.Namespace = pod.Namespace
				oPod.HostIP = pod.Status.HostIP
				oPod.PodIP = pod.Status.PodIP
				oPod.ContainerID = pod.Status.ContainerStatuses[0].ContainerID
				oPod.OfflineTime = time.Now().Format("2006-01-02 15:04:05")
				impl.WorkQueue.Add(oPod)
			},
		})
		klog.Infof("cluster name:%s AddEventHandler pod key to queue", cluster.Name)
	}
	return impl, nil
}

// isObserveNamespaces
func (c *offlinepodImpl) isObserveNamespaces(ns string) bool {
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

func (c *offlinepodImpl) Start(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.WorkQueue.ShutDown()
	wait.Until(c.runWorker, time.Second, stopCh)
	<-stopCh
	klog.Infof("Shutting down workers, name: %s", c.Name)
	return nil
}

func (c *offlinepodImpl) runWorker() {
	for c.processNextWorkItem() {
	}
}

func (c *offlinepodImpl) processNextWorkItem() bool {
	obj, shutdown := c.WorkQueue.Get()
	if obj == nil {
		// Sometimes the Queue gives us nil items when it starts up
		c.WorkQueue.Forget(obj)
	}

	if shutdown {
		return false
	}

	defer c.WorkQueue.Done(obj)

	var req *model.OfflinePod
	var ok bool

	if req, ok = obj.(*model.OfflinePod); !ok {
		c.WorkQueue.Forget(obj)
		return true
	}

	if err := c.reconciler(context.TODO(), req); err != nil {
		c.WorkQueue.AddRateLimited(req)
		klog.V(3).Infof("name: %s reconciler failed. err: %v", req.Name, err)
		return false
	}

	c.WorkQueue.Forget(req)
	return true
}

func (c *offlinepodImpl) GetOfflinePod() *model.OfflinePod {
	return c.OfflinePodPool.Get().(*model.OfflinePod)
}

func (c *offlinepodImpl) PutOfflinePod(b *model.OfflinePod) {
	b.Name = ""
	b.ClusterName = ""
	b.Namespace = ""
	b.AppName = ""
	b.HostIP = ""
	b.PodIP = ""
	b.ContainerID = ""
	b.OfflineTime = ""
	b.Labels = nil
	c.OfflinePodPool.Put(b)
}
