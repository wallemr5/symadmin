package resources

import (
	"context"

	"emperror.dev/errors"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources/patch"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DesiredState string

const (
	DesiredStatePresent DesiredState = "present"
	DesiredStateAbsent  DesiredState = "absent"
)

type Option struct {
	DesiredState     DesiredState
	IsRecreate       bool
	IsIgnoreReplicas bool
}

func prepareResourceForUpdate(current, desired runtime.Object) {
	switch desired.(type) {
	case *corev1.Service:
		svc := desired.(*corev1.Service)
		svc.Spec.ClusterIP = current.(*corev1.Service).Spec.ClusterIP
	}
}

func Reconcile(ctx context.Context, c client.Client, desired runtime.Object, opt Option) (int, error) {
	if opt.DesiredState == "" {
		opt.DesiredState = DesiredStatePresent
	}

	var change int
	var current = desired.DeepCopyObject()

	key, err := client.ObjectKeyFromObject(current)
	if err != nil {
		return change, err
	}

	name := key.String()
	err = c.Get(ctx, key, current)
	if err != nil && !apierrors.IsNotFound(err) {
		return change, errors.Wrapf(err, "getting resource name: %s", name)
	}
	if apierrors.IsNotFound(err) {
		if opt.DesiredState == DesiredStatePresent {
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(desired); err != nil {
				klog.Errorf("failed to set last applied annotation name: %s err: %+v", name, err)
			}
			if err := c.Create(ctx, desired); err != nil {
				return change, errors.Wrapf(err, "creating resource failed name: %s", name)
			}
			klog.Infof("resource name: %s created", name)
			change++
		}
	} else {
		if opt.DesiredState == DesiredStatePresent {
			var patchResult *patch.PatchResult
			calcOpts := []patch.CalculateOption{
				patch.IgnoreStatusFields(),
			}

			if utils.IsObjectLabelsChange(desired, current) {
				goto Update
			}

			if opt.IsIgnoreReplicas {
				switch desired.(type) {
				case *appsv1.Deployment:
					desiredDeploy := desired.(*appsv1.Deployment)
					currentDeploy := current.(*appsv1.Deployment)
					desiredReplicas := utils.GetWorkloadReplicas(desiredDeploy.Spec.Replicas)
					currentReplicas := utils.GetWorkloadReplicas(currentDeploy.Spec.Replicas)
					if desiredReplicas != 0 && currentReplicas > desiredReplicas {
						calcOpts = append(calcOpts, patch.IgnoreDeployReplicasFields())
					}
				case *appsv1.StatefulSet:
					desiredSta := desired.(*appsv1.StatefulSet)
					currentSta := current.(*appsv1.StatefulSet)
					desiredReplicas := utils.GetWorkloadReplicas(desiredSta.Spec.Replicas)
					currentReplicas := utils.GetWorkloadReplicas(currentSta.Spec.Replicas)
					if desiredReplicas != 0 && currentReplicas > desiredReplicas {
						calcOpts = append(calcOpts, patch.IgnoreStsReplicasFields())
					}
				}
			}

			patchResult, err = patch.DefaultPatchMaker.Calculate(current, desired, calcOpts...)
			if err != nil {
				klog.Errorf("could not match object name: %s err: %+v", name, err)
				return change, err
			} else if patchResult.IsEmpty() {
				klog.V(4).Infof("resource name: %s unchanged is in sync", name)
				return change, nil
			} else {
				klog.V(4).Infof("resource name: %s diffs patch: %s", name, string(patchResult.Patch))
			}

		Update:
			// Need to set this before resourceversion is set, as it would constantly change otherwise
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(desired); err != nil {
				klog.Errorf("Failed to set last applied annotation name: %s err: %+v", name, err)
			}

			if opt.IsRecreate {
				metaAccessor := meta.NewAccessor()
				currentResourceVersion, err := metaAccessor.ResourceVersion(current)
				if err != nil {
					klog.Errorf("name: %s metaAccessor err: %+v", name, err)
					return change, nil
				}

				metaAccessor.SetResourceVersion(desired, currentResourceVersion)
				prepareResourceForUpdate(current, desired)
				if err := c.Update(ctx, desired); err != nil {
					if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
						klog.Infof("resource name: %s needs to be re-created err: %+v", name, err)
						err := c.Delete(ctx, current)
						if err != nil {
							return change, errors.Wrapf(err, "could not delete resource name: %s", name)
						}
						klog.Infof("resource name: %s deleted", name)
						if err := c.Create(ctx, desired); err != nil {
							return change, errors.Wrapf(err, "creating resource name: %s", name)
						}
						klog.Infof("resource name: %s recreated", name)
						change++
						return change, nil
					}
					return change, errors.Wrapf(err, "updating resource name: %s", name)
				}
				klog.Infof("resource name: %s updated", name)
			} else {
				metaAccessor := meta.NewAccessor()
				retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					currentResourceVersion, err := metaAccessor.ResourceVersion(current)
					if err != nil {
						klog.Errorf("name: %s metaAccessor err: %+v", name, err)
						return err
					}

					metaAccessor.SetResourceVersion(desired, currentResourceVersion)
					prepareResourceForUpdate(current, desired)

					updateErr := c.Update(ctx, desired)
					if updateErr == nil {
						klog.V(4).Infof("updating resource name: %s successfully", name)
						return nil
					}

					// Get object again when updating is failed.
					getErr := c.Get(ctx, key, current)
					if getErr != nil {
						return errors.Wrapf(err, "updated get resource name: %s", name)
					}

					return updateErr
				})

				if retryErr != nil {
					klog.Errorf("name: %s retryErr: %+v", name, retryErr)
					return change, errors.Errorf("name: %s only update err: %+v, please check", name, retryErr)
				}
				change++
			}
		} else if opt.DesiredState == DesiredStateAbsent {
			if err := c.Delete(ctx, current); err != nil {
				return change, errors.Wrapf(err, "deleting resource name: %s", name)
			}
			klog.Infof("resource name: %s deleted successfully", name)
			change++
		}
	}
	return change, nil
}
