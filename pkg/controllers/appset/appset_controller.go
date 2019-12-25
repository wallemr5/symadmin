package appset

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	"gopkg.in/resty.v1"
	ctrl "sigs.k8s.io/controller-runtime"

	kruisev1alpha1 "github.com/openkruise/kruise/pkg/apis/apps/v1alpha1"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/customctrl"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Reconciler implements controller.Reconciler
type AppSetReconciler struct {
	manager.Manager
	DksMgr            *pkgmanager.DksManager
	SymServerRlsPath  string
	SymServerCfgPath  string
	LastReconcileTime time.Time
	MigratePeriod     time.Duration
	MigrateParallel   int

	AppSetIndexInformer cache.SharedIndexInformer

	Mx sync.RWMutex
}

func (r *AppSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workloadv1beta1.AppSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&kruisev1alpha1.StatefulSet{}).
		Owns(&corev1.Service{}).
		WithEventFilter(utils.GetWatchPredicateForNs()).
		WithEventFilter(utils.GetWatchPredicateForApp()).
		// Watches(&source.Kind{Type: &corev1.Pod{}}, &handler.Funcs{}).
		Watches(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: utils.GetEnqueueRequestsMapper()}).
		Complete(r)
}

func Add(mgr manager.Manager, cMgr *pkgmanager.DksManager) error {
	r, impl := NewAppSetController(mgr, cMgr)
	if r == nil {
		return fmt.Errorf("NewAppSetController err")
	}

	err := mgr.Add(impl)
	if err != nil {
		klog.Fatal("Can't add runnable for appset controller")
		return err
	}

	return nil
}

func NewAppSetController(mgr manager.Manager, cMgr *pkgmanager.DksManager) (*Reconciler, *customctrl.Impl) {
	c := &Reconciler{
		DksMgr:  cMgr,
		Manager: mgr,
	}

	cacher := mgr.GetCache()
	appSetInformer, err := cacher.GetInformer(&workloadv1beta1.AppSet{})
	if err != nil {
		klog.Errorf("cacher get informer err:%+v", err)
		return nil, nil
	}
	appSetLister := devopslistersv1.NewAppSetLister(appSetInformer.GetIndexer())
	c.AppSetLister = appSetLister
	c.AppSetIndexInformer = appSetInformer

	threadiness := 2
	impl := controller.NewImpl(c, controllerAgentName, nil, &threadiness, cluster.ObservedNamespace...)

	for name, k := range cMgr.Manager.ClusterCache {
		if !strings.Contains(k.Role, "worker") {
			continue
		}

		klog.Infof("Setting up event handlers, name: %s", name)
		k.DeploymentInformer.Informer().AddEventHandler(controller.HandlerWraps(impl.EnqueueMultiLabelOfCluster))
		k.StatefulSetInformer.Informer().AddEventHandler(controller.HandlerWraps(impl.EnqueueMultiLabelOfCluster))
		k.MigrateInformer.Informer().AddEventHandler(controller.HandlerWraps(impl.EnqueueMulti))
		if err := controller.StartInformers(
			k.StopChannel,
			k.StatefulSetInformer.Informer(),
			k.DeploymentInformer.Informer(),
			k.MigrateInformer.Informer(),
		); err != nil {
			klog.Error("Failed to start informers", err)
			return nil, nil
		}
	}

	c.AppSetIndexInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: impl.Enqueue,
		UpdateFunc: func(old, cur interface{}) {
			curApp := cur.(*devopsv1.AppSet)
			oldApp := old.(*devopsv1.AppSet)
			if curApp.Spec.IsCustom == oldApp.Spec.IsCustom &&
				curApp.Spec.AllocRevisionSpec.Enable == oldApp.Spec.AllocRevisionSpec.Enable {
				return
			}
			impl.Enqueue(cur)
		},
		DeleteFunc: impl.Enqueue,
	})

	cMgr.HealthHander.AddReadinessCheck("appset_cache_sync",
		func() error {
			if c.AppSetIndexInformer.HasSynced() {
				return nil
			}
			return fmt.Errorf("appset cache not sync")
		})

	c.SymServerRlsPath = viper.GetString(config.SymServerPrefix) + viper.GetString(config.SymServerAddr)
	c.SymServerCfgPath = viper.GetString(config.SymServerPrefix) + viper.GetString(config.SymServerAppClusters)
	c.MigratePeriod = viper.GetDuration(config.MigratePeriod) * time.Second
	c.MigrateParallel = viper.GetInt(config.MigrateParallel)
	klog.Infof("MigratePeriod:%+v, MigrateParallel:%d", c.MigratePeriod, c.MigrateParallel)

	client := resty.New()
	// client.SetDebug(true)
	client.SetDebug(false)
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("User-Agent", controllerAgentName)
	client.SetRetryCount(3).SetRetryWaitTime(100 * time.Millisecond).SetRetryMaxWaitTime(100 * time.Millisecond)
	c.Client = client

	dbSyncer := trigger.NewTriggerDb(cMgr.Manager, appSetInformer)
	c.DbSyncer = dbSyncer
	c.DbSyncer.AddActor(func(spec *trigger.TriggerRevSpec) {
		c.Mx.Lock()
		defer c.Mx.Unlock()

		c.RevSpec.RevisionId = spec.RevisionId
		c.RevSpec.Enable = spec.Enable
		c.RevSpec.Policy = spec.Policy

		// Less than 30s， can't modify MigratePeriod
		if viper.GetDuration(config.MigratePeriod) >= 30 {
			switch spec.Policy {
			case 0:
				c.MigratePeriod = 30 * time.Second
			case 1:
				c.MigratePeriod = 1 * time.Minute
			case 2:
				c.MigratePeriod = 2 * time.Minute
			case 3:
				c.MigratePeriod = 5 * time.Minute
			case 4:
				c.MigratePeriod = 10 * time.Minute
			default:
				c.MigratePeriod = 30 * time.Second
			}
		}

		klog.V(3).Infof("Enable:%d, MigratePeriod:%+v, MigrateParallel:%d",
			c.RevSpec.Enable, c.MigratePeriod, c.MigrateParallel)
	})

	err = mgr.Add(dbSyncer)
	if err != nil {
		klog.Fatal("Can't add runnable for dbSyncer")
	}

	err = mgr.Add(NewPolicyTrigger(c))
	if err != nil {
		klog.Fatal("Can't add runnable for PolicyTrigger")
	}

	cMgr.Router.AddRoutes(controllerAgentName, c.Routes())
	c.Impl = impl
	return c, impl
}
