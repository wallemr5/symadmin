package svc

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/goph/emperror"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	componentName = "svc"
)

type Reconciler struct {
	resources.Reconciler
	// other
	port int
}

func New(mgr manager.Manager, config *workloadv1beta1.AdvDeployment, port int) *Reconciler {
	return &Reconciler{
		Reconciler: resources.Reconciler{
			Mgr:    mgr,
			Config: config,
		},
		port: port,
	}
}
func (r *Reconciler) Service() runtime.Object {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Config.Spec.ServiceName,
			Namespace: r.Config.Namespace,
			Labels:    r.GetSvcLabels(),
			// Annotations: nil,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(r.port),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				utils.ObserveMustLabelAppName: r.Config.Name,
			},
		},
	}

	_ = controllerutil.SetControllerReference(r.Config, svc, r.Mgr.GetScheme())
	return svc
}

func (r *Reconciler) Deployment() runtime.Object {
	return &appsv1.Deployment{}
}

func (r *Reconciler) Reconcile(log logr.Logger) error {
	log = log.WithValues("component", componentName)

	for _, res := range []resources.Resource{
		r.Service,
		// r.Deployment,
	} {
		o := res()
		result, err := controllerutil.CreateOrUpdate(context.TODO(), r.Mgr.GetClient(), o, func() error {
			return nil
		})
		if err != nil {
			return emperror.WrapWith(err, "failed to reconcile resource",
				"resource", o.GetObjectKind().GroupVersionKind(),
				"result", result)
		}
	}
	log.Info("Reconciled")
	return nil
}
