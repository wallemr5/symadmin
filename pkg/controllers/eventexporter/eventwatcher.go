package eventexporter

import (
	"context"
	"time"

	"fmt"

	"github.com/go-logr/logr"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/eventexporter/exporter"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/eventexporter/kube"
	pkgLabels "gitlab.dmall.com/arch/sym-admin/pkg/labels"
	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "events-controller"
)

const (
	EventConfig = `
route:
  routes:
    - drop:
        - namespace: "*test*"
        - type: "Normal"
      match:
        - receiver: "alert"
receivers:
  - name: "alert"
    alertmanager:
      headers:
        User-Agent: "event-exporter"
`
)

type eventReconciler struct {
	client.Client
	Name            string
	Log             logr.Logger
	Mgr             manager.Manager
	engine          *exporter.Engine
	labelCache      *kube.LabelCache
	annotationCache *kube.AnnotationCache
	ClusterName     string
}

func onEvent(obj interface{}) bool {
	e := obj.(*corev1.Event)
	if e.Type == corev1.EventTypeNormal {
		return false
	}

	if time.Now().Sub(e.CreationTimestamp.Time) > 1*time.Minute {
		return false
	}

	return true
}

func GetWatchPredicateForEvent() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return onEvent(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return onEvent(e.ObjectNew)
		},
	}
}

func Add(mgr manager.Manager, cMgr *pkgmanager.DksManager) error {
	r := &eventReconciler{
		Name:   controllerName,
		Client: mgr.GetClient(),
		Mgr:    mgr,
		Log:    ctrl.Log.WithName("controllers").WithName("events"),
	}

	cfg := mgr.GetConfig()
	dynClient := dynamic.NewForConfigOrDie(cfg)
	kubeCli := kubernetes.NewForConfigOrDie(cfg)

	// Create a new runtime controller for events
	ctl, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r, MaxConcurrentReconciles: cMgr.Opt.Threadiness})
	if err != nil {
		r.Log.Error(err, "Creating a new event controller has an error")
		return err
	}

	// Watch for changes to events for runtime controller
	err = ctl.Watch(&source.Kind{Type: &corev1.Event{}}, &handler.EnqueueRequestForObject{}, GetWatchPredicateForEvent())
	if err != nil {
		r.Log.Error(err, "Watching event has an error")
		return err
	}

	eventCfg := &exporter.Config{}
	err = yaml.Unmarshal([]byte(EventConfig), eventCfg)
	if err != nil {
		klog.Fatalf("cannot parse config err: %+v, yaml: \n%s", err, EventConfig)
	}

	for i := range eventCfg.Receivers {
		receiver := &eventCfg.Receivers[i]
		if receiver.AlertManager != nil {
			if len(cMgr.Opt.AlertEndpoint) > 0 {
				receiver.AlertManager.Endpoint = cMgr.Opt.AlertEndpoint
			} else {
				list, err := kubeCli.CoreV1().Services("monitoring").List(context.TODO(),
					metav1.ListOptions{LabelSelector: fmt.Sprintf("app=%s", "prometheus-operator-alertmanager")})
				if err != nil {
					klog.Fatalf("get alertmanager service err: %+v", err)
				}

				receiver.AlertManager.Endpoint = fmt.Sprintf("http://%s.%s.svc:9093", list.Items[0].Name, list.Items[0].Namespace)
			}
		}
	}
	r.engine = exporter.NewEngine(eventCfg)
	r.labelCache = kube.NewLabelCache(kubeCli, dynClient)
	r.annotationCache = kube.NewAnnotationCache(kubeCli, dynClient)
	return nil
}

func (r *eventReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logger := r.Log.WithValues("key", req.NamespacedName.String())

	startTime := time.Now()
	defer func() {
		diffTime := time.Since(startTime)
		var logLevel klog.Level
		if diffTime > 1*time.Second {
			logLevel = 2
		} else if diffTime > 100*time.Millisecond {
			logLevel = 4
		} else {
			logLevel = 5
		}
		klog.V(logLevel).Infof("##### [%s] reconciling is finished. time taken: %v. ", req.NamespacedName.String(), diffTime)
	}()

	e := &corev1.Event{}
	err := r.Client.Get(ctx, req.NamespacedName, e)
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.V(3).Infof("not find event with name: %s, skip", req.NamespacedName.String())
			return reconcile.Result{}, nil
		}

		logger.Error(err, "failed to get event")
		return reconcile.Result{}, err
	}

	logger.Info("Received event", "namespace", e.Namespace,
		"reason", e.Reason, "involvedObject", e.InvolvedObject.Name, "msg", e.Message)

	ev := &kube.EnhancedEvent{
		Event: *e,
	}

	labels, err := r.labelCache.GetLabelsWithCache(&e.InvolvedObject)
	if err != nil {
		logger.Error(err, "Cannot list labels of the object")
	} else {
		ev.InvolvedObject.Labels = labels
		ev.InvolvedObject.ObjectReference = e.InvolvedObject
	}

	annotations, err := r.annotationCache.GetAnnotationsWithCache(&e.InvolvedObject)
	if err != nil {
		logger.Error(err, "Cannot list annotations of the object")
	} else {
		ev.InvolvedObject.Annotations = annotations
		ev.InvolvedObject.ObjectReference = e.InvolvedObject
	}

	if len(r.ClusterName) == 0 {
		if name, ok := ev.InvolvedObject.Labels[pkgLabels.ObserveMustLabelClusterName]; ok {
			r.ClusterName = name
		}
	}

	ev.ClusterName = r.ClusterName
	r.engine.Route.ProcessEvent(ev, r.engine.Registry)
	return reconcile.Result{}, nil
}
