package advdeployment

import (
	"context"

	"github.com/pkg/errors"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	hapirelease "k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Remove the sym-admin's finalizer from the entry list.
func (r *AdvDeploymentReconciler) RemoveFinalizers(ctx context.Context, req ctrl.Request) error {
	obj := &workloadv1beta1.AdvDeployment{}
	err := r.Client.Get(ctx, req.NamespacedName, obj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.V(3).Infof("Can not find any advDeploy with name %s, ignore it.", req.NamespacedName)
			return nil
		}

		klog.Errorf("failed to get AdvDeployment, err: %v", err)
		return err
	}

	if utils.ContainsString(obj.ObjectMeta.Finalizers, labels.ControllerFinalizersName) {
		var i int
		for idx, fz := range obj.ObjectMeta.Finalizers {
			if fz == labels.ControllerFinalizersName {
				i = idx
				break
			}
		}
		obj.ObjectMeta.Finalizers = append(obj.ObjectMeta.Finalizers[:i], obj.ObjectMeta.Finalizers[i+1:]...)
		if err := r.Client.Update(ctx, obj); err != nil {
			klog.Errorf("Can not remove the finalizer entry from the list, err: %v", err)
			return errors.Wrap(err, "Can not remove the finalizer entry from the list")
		}
		klog.V(3).Infof("advDeploy: [%s], Remove the Finalizers and update it successfully", obj.Name)
	}

	return nil
}

// Delete all releases of this advDeployment
func (r *AdvDeploymentReconciler) CleanAllReleases(advDeploy *workloadv1beta1.AdvDeployment) error {
	hClient, err := helmv2.NewClientFromConfig(r.Cfg, r.KubeCli, "")
	if err != nil {
		klog.Errorf("New helm Clinet err:%+v", err)
		return err
	}
	defer hClient.Close()

	response, err := helmv2.ListReleases(labels.MakeHelmReleaseFilter(advDeploy.Name), hClient)
	if err != nil || response == nil {
		klog.Errorf("no find helm releases by name: %s", advDeploy.Name)
		return err
	}

	for _, helmRls := range response.Releases {
		if err := helmv2.DeleteRelease(helmRls.Name, hClient); err != nil {
			klog.Errorf("DeleteRelease name: %s, err:%+v", helmRls.Name, err)
			return err
		}

		klog.V(4).Infof("rlsName: %s clean successed", helmRls.Name)
	}

	return nil
}

// We try to converge the advDeploy's status to that desired status.
func (r *AdvDeploymentReconciler) ApplyReleases(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment) error {
	// Initialize a new helm client
	hClient, err := helmv2.NewClientFromConfig(r.Cfg, r.KubeCli, "")
	if err != nil {
		klog.Errorf("Initializing a new helm clinet has an error: %+v", err)
		return err
	}
	defer hClient.Close()

	/*
	  Find out all releases belong to this application to
	*/
	response, err := helmv2.ListReleases(labels.MakeHelmReleaseFilter(advDeploy.Name), hClient)
	if err != nil {
		klog.Errorf("Can not find a releases with name: %s, err: %v", advDeploy.Name, err)
		return err
	}

	// The releases which are not defined in specification but are running already.
	var redundantReleases []string

	if response != nil {
		runningReleases := response.Releases
		for _, rr := range runningReleases {
			if isRedundantRelease(rr.Name, advDeploy) {
				redundantReleases = append(redundantReleases, rr.Name)
			}

			// Delete the release as long as we found its status code is not correct
			if rr.Info.Status.GetCode() == hapirelease.Status_DELETED || rr.Info.Status.GetCode() == hapirelease.Status_FAILED || rr.Info.Status.GetCode() == hapirelease.Status_UNKNOWN {
				klog.Infof("Release: [%s]'s status mean there may be some problem, description: %s", rr.Name, rr.Info.Description)

				// Delete this release with purge flag.
				if err := helmv2.DeleteRelease(rr.Name, hClient); err != nil {
					klog.Errorf("Delete the release due to its status, but has an error, rls name: %s, err: %+v", rr.Name, err)
					return err
				}

				klog.V(4).Infof("rlsName: [%s], Delete the release due to its status", rr.Name)
			}
		}
	} else {
		klog.V(4).Infof("Listing all releases may be has an error, there is no response feed back.")
	}

	var (
		specChartUrlName    string
		specChartUrlVersion string
		specRawChart        []byte
		// rls    *hapirelease.Release

	)

	if advDeploy.Spec.PodSpec.Chart.RawChart != nil {
		specRawChart = *advDeploy.Spec.PodSpec.Chart.RawChart
	}

	if advDeploy.Spec.PodSpec.Chart.ChartUrl != nil {
		specChartUrlName = advDeploy.Spec.PodSpec.Chart.ChartUrl.Url
		specChartUrlVersion = advDeploy.Spec.PodSpec.Chart.ChartUrl.ChartVersion
	}
	for _, podSet := range advDeploy.Spec.Topology.PodSets {
		_, err := helmv2.ApplyRelease(podSet.Name, specChartUrlName, specChartUrlVersion, specRawChart,
			hClient, r.HelmEnv.Helmv2env, advDeploy.Namespace, findRunningReleases(podSet.Name, response), []byte(podSet.RawValues))
		if err != nil {
			klog.Errorf("Podset: [%s], applying the release has an error: %v", podSet.Name, err)
			return err
		}
	}

	// We must to clean all the releases which are not defined in specification.
	for _, rlsName := range redundantReleases {
		if err := helmv2.DeleteRelease(rlsName, hClient); err != nil {
			klog.Errorf("Deleting redundant release[%s] has an error: %+v", rlsName, err)
			return err
		}

		klog.V(4).Infof("deleting redundant release[%s] successfully.", rlsName)
	}

	return nil
}

func isRedundantRelease(rlsName string, advDeploy *workloadv1beta1.AdvDeployment) bool {
	for _, podSet := range advDeploy.Spec.Topology.PodSets {
		if podSet.Name == rlsName {
			return false
		}
	}

	return true
}

// Find the release with its name from running release response.
func findRunningReleases(name string, rlsList *rls.ListReleasesResponse) *hapirelease.Release {
	if rlsList == nil {
		return nil
	}

	for _, r := range rlsList.Releases {
		if r.Name == name {
			if r.Info.Status.GetCode() != hapirelease.Status_DEPLOYED {
				return nil
			}
			return r
		}
	}

	return nil
}
