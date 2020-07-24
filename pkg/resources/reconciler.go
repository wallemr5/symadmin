package resources

import (
	"context"

	"reflect"

	"github.com/pkg/errors"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources/patch"
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
		return change, errors.Wrapf(err, "copy key[%s]", key)
	}

	err = c.Get(ctx, key, current)
	if err != nil && !apierrors.IsNotFound(err) {
		return change, errors.Wrapf(err, "getting resource key[%s]", key)
	}
	if apierrors.IsNotFound(err) {
		if opt.DesiredState == DesiredStatePresent {
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(desired); err != nil {
				klog.Errorf("Failed to set last applied annotation key[%s] err: %v", key, err)
			}
			if err := c.Create(ctx, desired); err != nil {
				return change, errors.Wrapf(err, "creating resource failed key[%s]", key)
			}
			klog.Infof("resource key[%s] created", key)
			change++
		}
	} else {
		if opt.DesiredState == DesiredStatePresent {
			var patchResult *patch.PatchResult
			calcOpts := []patch.CalculateOption{
				patch.IgnoreStatusFields(),
			}

			if svcDesired, ok := desired.(*corev1.Service); ok {
				svcCurrent, _ := current.(*corev1.Service)
				if !reflect.DeepEqual(svcCurrent.GetLabels(), svcDesired.GetLabels()) {
					klog.Infof("type svc name[%s] labels not same", svcDesired.Name)
					goto Update
				}
			}

			if _, ok := desired.(*appsv1.Deployment); ok && opt.IsIgnoreReplicas {
				calcOpts = append(calcOpts, patch.IgnoreDeployReplicasFields())
			}

			if _, ok := desired.(*appsv1.StatefulSet); ok && opt.IsIgnoreReplicas {
				calcOpts = append(calcOpts, patch.IgnoreStsReplicasFields())
			}

			patchResult, err = patch.DefaultPatchMaker.Calculate(current, desired, calcOpts...)
			if err != nil {
				klog.Errorf("could not match object key[%s] err: %v", key, err)
				return change, err
			} else if patchResult.IsEmpty() {
				klog.V(4).Infof("resource key[%s] unchanged is in sync", key)
				return change, nil
			} else {
				klog.V(2).Infof("resource key[%s] diffs patch: %s", key, string(patchResult.Patch))
			}

		Update:
			// Need to set this before resourceversion is set, as it would constantly change otherwise
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(desired); err != nil {
				klog.Errorf("Failed to set last applied annotation key[%s] err: %v", key, err)
			}

			if opt.IsRecreate {
				metaAccessor := meta.NewAccessor()
				currentResourceVersion, err := metaAccessor.ResourceVersion(current)
				if err != nil {
					klog.Errorf("key[%s] metaAccessor err: %v", key, err)
					return change, nil
				}

				metaAccessor.SetResourceVersion(desired, currentResourceVersion)
				prepareResourceForUpdate(current, desired)
				if err := c.Update(ctx, desired); err != nil {
					if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
						klog.Infof("resource key[%s] needs to be re-created err: %v", key, err)
						err := c.Delete(ctx, current)
						if err != nil {
							return change, errors.Wrapf(err, "could not delete resource key[%s]", key)
						}
						klog.Infof("resource key[%s] deleted", key)
						if err := c.Create(ctx, desired); err != nil {
							return change, errors.Wrapf(err, "creating resource key[%s]", key)
						}
						klog.Infof("resource key[%s] created", key)
						change++
						return change, nil
					}
					return change, errors.Wrapf(err, "updating resource key[%s]", key)
				}
				klog.Infof("resource key:%s updated", key)
			} else {
				metaAccessor := meta.NewAccessor()
				retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
					currentResourceVersion, err := metaAccessor.ResourceVersion(current)
					if err != nil {
						klog.Errorf("key[%s] metaAccessor err: %v", key, err)
						return err
					}

					metaAccessor.SetResourceVersion(desired, currentResourceVersion)
					prepareResourceForUpdate(current, desired)

					updateErr := c.Update(ctx, desired)
					if updateErr == nil {
						klog.V(2).Infof("Updating resource key[%s] successfully", key)
						return nil
					}

					// Get object again when updating is failed.
					getErr := c.Get(ctx, key, current)
					if getErr != nil {
						return errors.Wrapf(err, "updated get resource key[%s]", key)
					}

					return updateErr
				})

				if retryErr != nil {
					klog.Errorf("key[%s] retryErr: %v", key, retryErr)
					return change, errors.Errorf("key[%s] only update err: %v, please check", key, retryErr)
				}
				change++
			}
		} else if opt.DesiredState == DesiredStateAbsent {
			if err := c.Delete(ctx, current); err != nil {
				return change, errors.Wrapf(err, "deleting resource key[%s]", key)
			}
			klog.Infof("resource key[%s] deleted successfully", key)
			change++
		}
	}
	return change, nil
}
