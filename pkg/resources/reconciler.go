package resources

import (
	"context"

	"github.com/pkg/errors"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources/patch"
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

func prepareResourceForUpdate(current, desired runtime.Object) {
	switch desired.(type) {
	case *corev1.Service:
		svc := desired.(*corev1.Service)
		svc.Spec.ClusterIP = current.(*corev1.Service).Spec.ClusterIP
	}
}

func Reconcile(ctx context.Context, c client.Client, desired runtime.Object, desiredState DesiredState, isRecreate bool) error {
	if desiredState == "" {
		desiredState = DesiredStatePresent
	}

	var current = desired.DeepCopyObject()
	// desiredType := reflect.TypeOf(desired)
	// var desiredCopy = desired.DeepCopyObject()
	key, err := client.ObjectKeyFromObject(current)
	if err != nil {
		return errors.Wrapf(err, "copy key[%s]", key)
	}

	err = c.Get(ctx, key, current)
	if err != nil && !apierrors.IsNotFound(err) {
		return errors.Wrapf(err, "getting resource key[%s]", key)
	}
	if apierrors.IsNotFound(err) {
		if desiredState == DesiredStatePresent {
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(desired); err != nil {
				klog.Errorf("Failed to set last applied annotation key[%s] err: %v", key, err)
			}
			if err := c.Create(ctx, desired); err != nil {
				return errors.Wrapf(err, "creating resource failed key[%s]", key)
			}
			klog.Infof("resource key[%s] created", key)
		}
	} else {
		if desiredState == DesiredStatePresent {
			patchResult, err := patch.DefaultPatchMaker.Calculate(current, desired)
			if err != nil {
				klog.Errorf("could not match object key[%s] err: %v", key, err)
			} else if patchResult.IsEmpty() {
				klog.V(4).Infof("resource key[%s] unchanged is in sync", key)
				return nil
			} else {
				klog.V(2).Infof("resource key[%s] diffs patch: %s", key, string(patchResult.Patch))
			}

			// Need to set this before resourceversion is set, as it would constantly change otherwise
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(desired); err != nil {
				klog.Errorf("Failed to set last applied annotation key[%s] err: %v", key, err)
			}

			if isRecreate {
				if err := c.Update(ctx, desired); err != nil {
					if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
						klog.Infof("resource key[%s] needs to be re-created err: %v", key, err)
						err := c.Delete(ctx, current)
						if err != nil {
							return errors.Wrapf(err, "could not delete resource key[%s]", key)
						}
						klog.Infof("resource key[%s] deleted", key)
						if err := c.Create(ctx, desired); err != nil {
							return errors.Wrapf(err, "creating resource key[%s]", key)
						}
						klog.Infof("resource key[%s] created", key)
						return nil
					}
					return errors.Wrapf(err, "updating resource key[%s]", key)
				}
				klog.Infof("resource key:%s updated", key)
			} else {
				metaAccessor := meta.NewAccessor()
				err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
					currentResourceVersion, err := metaAccessor.ResourceVersion(current)
					if err != nil {
						return err
					}

					metaAccessor.SetResourceVersion(desired, currentResourceVersion)
					prepareResourceForUpdate(current, desired)

					updateErr := c.Update(ctx, desired)
					if updateErr == nil {
						klog.V(2).Infof("Updating resource key[%s] successfully", key)
						return nil
					}

					// Get the advdeploy again when updating is failed.
					getErr := c.Get(ctx, key, current)
					if getErr != nil {
						return errors.Wrapf(err, "updated get resource key[%s]", key)
					}

					return updateErr
				})
			}

		} else if desiredState == DesiredStateAbsent {
			if err := c.Delete(ctx, current); err != nil {
				return errors.Wrapf(err, "deleting resource key[%s]", key)
			}
			klog.Infof("resource key[%s] deleted successfully", key)
		}
	}
	return nil
}
