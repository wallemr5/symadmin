package resources

import (
	"fmt"
	"github.com/go-logr/logr"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"context"
	"github.com/goph/emperror"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources/patch"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type DesiredState string

const (
	DesiredStatePresent DesiredState = "present"
	DesiredStateAbsent  DesiredState = "absent"
)

type Reconciler struct {
	Mgr    manager.Manager
	Config *workloadv1beta1.AdvDeployment
}

type ComponentReconciler interface {
	Reconcile(log logr.Logger) error
}

type Resource func() runtime.Object

func (r *Reconciler) GetSvcLabels() map[string]string {
	var domain string
	if r.Config.Spec.Domain != nil {
		domain = *r.Config.Spec.Domain
	} else {
		domain = fmt.Sprintf("%s.dmalll.com", r.Config.Name)
	}
	labels := map[string]string{
		utils.ObserveMustLabelAppName:          r.Config.Name,
		utils.ObserveMustLabelLightningDomain0: domain,
	}
	return utils.MergeLabels(labels, r.Config.Spec.Strategy.Meta)
}

func (r *Reconciler) GetDeployLabels(name string) map[string]string {
	ldcName, groupName, _ := utils.SplitMetaLdcGroupKey(name)

	labels := map[string]string{
		utils.ObserveMustLabelAppName:     r.Config.Name,
		utils.ObserveMustLabelReleaseName: r.Config.Name + "-" + name,
		utils.ObserveMustLabelLdcName:     ldcName,
		utils.ObserveMustLabelGroupName:   groupName,
	}

	return utils.MergeLabels(labels, r.Config.Spec.Strategy.Meta)
}

func (r *Reconciler) GetAffinity() *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 1,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								utils.ObserveMustLabelAppName: r.Config.Name,
							},
						},
						Namespaces:  []string{r.Config.Namespace},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}
}

func prepareResourceForUpdate(current, desired runtime.Object) {
	switch desired.(type) {
	case *corev1.Service:
		svc := desired.(*corev1.Service)
		svc.Spec.ClusterIP = current.(*corev1.Service).Spec.ClusterIP
	}
}

func Reconcile(log logr.Logger, c client.Client, desired runtime.Object, desiredState DesiredState) error {
	if desiredState == "" {
		desiredState = DesiredStatePresent
	}

	desiredType := reflect.TypeOf(desired)
	var current = desired.DeepCopyObject()
	var desiredCopy = desired.DeepCopyObject()
	key, err := client.ObjectKeyFromObject(current)
	if err != nil {
		return emperror.With(err, "kind", desiredType)
	}
	log = log.WithValues("kind", desiredType, "name", key.Name)

	err = c.Get(context.TODO(), key, current)
	if err != nil && !apierrors.IsNotFound(err) {
		return emperror.WrapWith(err, "getting resource failed", "kind", desiredType, "name", key.Name)
	}
	if apierrors.IsNotFound(err) {
		if desiredState == DesiredStatePresent {
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(desired); err != nil {
				log.Error(err, "Failed to set last applied annotation", "desired", desired)
			}
			if err := c.Create(context.TODO(), desired); err != nil {
				return emperror.WrapWith(err, "creating resource failed", "kind", desiredType, "name", key.Name)
			}
			log.Info("resource created")
		}
	} else {
		if desiredState == DesiredStatePresent {
			patchResult, err := patch.DefaultPatchMaker.Calculate(current, desired)
			if err != nil {
				log.Error(err, "could not match objects", "kind", desiredType, "name", key.Name)
			} else if patchResult.IsEmpty() {
				log.V(1).Info("resource is in sync")
				return nil
			} else {
				log.V(1).Info("resource diffs",
					"patch", string(patchResult.Patch),
					"current", string(patchResult.Current),
					"modified", string(patchResult.Modified),
					"original", string(patchResult.Original))
			}

			// Need to set this before resourceversion is set, as it would constantly change otherwise
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(desired); err != nil {
				log.Error(err, "Failed to set last applied annotation", "desired", desired)
			}

			metaAccessor := meta.NewAccessor()
			currentResourceVersion, err := metaAccessor.ResourceVersion(current)
			if err != nil {
				return err
			}

			metaAccessor.SetResourceVersion(desired, currentResourceVersion)
			prepareResourceForUpdate(current, desired)

			if err := c.Update(context.TODO(), desired); err != nil {
				if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
					log.Info("resource needs to be re-created", "error", err)
					err := c.Delete(context.TODO(), current)
					if err != nil {
						return emperror.WrapWith(err, "could not delete resource", "kind", desiredType, "name", key.Name)
					}
					log.Info("resource deleted")
					if err := c.Create(context.TODO(), desiredCopy); err != nil {
						return emperror.WrapWith(err, "creating resource failed", "kind", desiredType, "name", key.Name)
					}
					log.Info("resource created")
					return nil
				}

				return emperror.WrapWith(err, "updating resource failed", "kind", desiredType, "name", key.Name)
			}
			log.Info("resource updated")
		} else if desiredState == DesiredStateAbsent {
			if err := c.Delete(context.TODO(), current); err != nil {
				return emperror.WrapWith(err, "deleting resource failed", "kind", desiredType, "name", key.Name)
			}
			log.Info("resource deleted")
		}
	}
	return nil
}
