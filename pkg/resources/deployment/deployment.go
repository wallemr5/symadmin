package deployment

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/goph/emperror"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources/templates"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	componentName = "deploy"
)

type Reconciler struct {
	resources.Reconciler
	// other

}

func New(mgr manager.Manager, config *workloadv1beta1.AdvDeployment) *Reconciler {
	return &Reconciler{
		Reconciler: resources.Reconciler{
			Mgr:    mgr,
			Config: config,
		},
	}
}

func (r *Reconciler) Deployment(cell *workloadv1beta1.CellReplicas) runtime.Object {
	lb := r.GetDeployLabels(cell.CellName)

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Config.Name + "-" + cell.CellName,
			Namespace: r.Config.Namespace,
			Labels:    r.GetSvcLabels(),
			// Annotations: nil,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: utils.IntPointer(cell.Replicas),
			Strategy: templates.DefaultRecreateStrategy(),
			Selector: &metav1.LabelSelector{
				MatchLabels: lb,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      lb,
					Annotations: templates.DefaultDeployAnnotations(),
				},
				Spec: r.Config.Spec.Template.Spec,
			},
		},
	}

	if deploy.Spec.Template.Spec.Affinity == nil {
		deploy.Spec.Template.Spec.Affinity = r.GetAffinity()
	}

	_ = controllerutil.SetControllerReference(r.Config, deploy, r.Mgr.GetScheme())
	return deploy
}

func (r *Reconciler) DeploymentAll() []runtime.Object {
	var objs []runtime.Object

	for _, rs := range r.Config.Spec.Strategy.CellReplicas {
		objs = append(objs, r.Deployment(rs))
	}
	return objs
}

func (r *Reconciler) Reconcile(log logr.Logger) error {
	log = log.WithValues("component", componentName)

	cli := r.Mgr.GetClient()
	deploylist := &appsv1.DeploymentList{}
	err := cli.List(context.Background(), deploylist, client.InNamespace(r.Config.Namespace), client.MatchingLabels{"app": r.Config.Name})
	if err != nil {
		log.Error(err, "list", "name", r.Config.Name)
		return err
	}

	if len(deploylist.Items) == 0 {
		log.Info("maybe first deploy")
	}
	for _, deploy := range r.DeploymentAll() {
		err := resources.Reconcile(log, r.Mgr.GetClient(), deploy, resources.DesiredStatePresent)
		// result, err := controllerutil.CreateOrUpdate(context.TODO(), r.Mgr.GetClient(), deploy, func() error {
		// 	return nil
		// })
		if err != nil {
			return emperror.WrapWith(err, "failed to reconcile resource", "resource", deploy.GetObjectKind().GroupVersionKind())
		}
	}
	log.Info("Reconciled")
	return nil
}
