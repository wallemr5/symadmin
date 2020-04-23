package advdeployment

import (
	"context"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	hapirelease "k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/klog"
)

// CleanAllReleases delete all releases of this advDeployment
func (r *AdvDeploymentReconciler) CleanAllReleases(advDeploy *workloadv1beta1.AdvDeployment) error {
	hClient, err := helmv2.NewClientFromConfig(r.Cfg, r.KubeCli, "", r.HelmEnv.Helmv2env)
	if err != nil {
		klog.Errorf("Initializing a helm client has an error: %+v", err)
		return err
	}
	defer hClient.Close()

	response, err := helmv2.ListReleases(labels.MakeHelmReleaseFilter(advDeploy.Name), hClient)
	if err != nil || response == nil {
		klog.Errorf("Can not find release[%s] before deleting it", advDeploy.Name)
		return err
	}

	for _, helmRls := range response.Releases {
		if err := helmv2.DeleteRelease(helmRls.Name, hClient); err != nil {
			klog.Errorf("Deleting release[%s] has an error:%+v", helmRls.Name, err)
			return err
		}

		klog.V(4).Infof("Release[%s] has been cleaned (or purge deleted) successfully", helmRls.Name)
	}

	return nil
}

// ApplyReleases we try to converge the advDeploy's status to that desired status.
// It returns true mean that we modified running releases in some way, otherwise no action happens on running releases.
func (r *AdvDeploymentReconciler) ApplyReleases(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment) (bool, error) {
	hasModifiedRls := false
	// Initialize a new helm client
	hClient, err := helmv2.NewClientFromConfig(r.Cfg, r.KubeCli, "k8s", r.HelmEnv.Helmv2env)
	if err != nil {
		klog.Errorf("Initializing a new helm clinet has an error: %+v", err)
		return hasModifiedRls, err
	}
	defer hClient.Close()

	/*
	  Find out all releases belong to this application to
	*/
	response, err := helmv2.ListReleases(labels.MakeHelmReleaseFilter(advDeploy.Name), hClient)
	if err != nil {
		klog.Errorf("Can not find all releases for advDeploy[%s] before applying them, err: %v", advDeploy.Name, err)
		return hasModifiedRls, err
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
				klog.Errorf("Release[%s]'s status means that there may be some problems here, description: %s", rr.Name, rr.Info.Description)
				r.recorder.Event(advDeploy, corev1.EventTypeWarning, "Release status may be some problems", rr.Info.Description)

				// Delete this release with purge flag.
				if err := helmv2.DeleteRelease(rr.Name, hClient); err != nil {
					klog.Errorf("Delete the release due to its status, but it has an error here, rls name: %s, err: %+v", rr.Name, err)
					return hasModifiedRls, err
				}
				hasModifiedRls = true

				klog.V(4).Infof("release[%s], Delete the release due to its status", rr.Name)
			}
		}
	} else {
		klog.V(4).Infof("Listing all releases may be has an error, there is no response feeds back.")
	}

	for _, podSet := range advDeploy.Spec.Topology.PodSets {
		specChartURLName, specChartURLVersion, specRawChart := getChartInfo(podSet, advDeploy)

		appliedRls, err := helmv2.ApplyRelease(podSet.Name, specChartURLName, specChartURLVersion, specRawChart,
			hClient, advDeploy.Namespace, findRunningReleases(podSet.Name, response), []byte(podSet.RawValues))
		if appliedRls != nil {
			hasModifiedRls = true
		}
		if err != nil {
			klog.Errorf("Podset: [%s], applying the release has an error: %v", podSet.Name, err)
			r.recorder.Event(advDeploy, corev1.EventTypeWarning, "Applying the release has an error", err.Error())
			return hasModifiedRls, err
		}
	}

	// We must to clean all the releases which are not defined in specification.
	for _, rlsName := range redundantReleases {
		if err := helmv2.DeleteRelease(rlsName, hClient); err != nil {
			klog.Errorf("Deleting redundant release[%s] has an error: %+v", rlsName, err)
			r.recorder.Event(advDeploy, corev1.EventTypeWarning, "Deleteing redundant release error", err.Error())
			return hasModifiedRls, err
		}
		hasModifiedRls = true
		klog.V(4).Infof("deleting redundant release[%s] successfully.", rlsName)
	}

	return hasModifiedRls, nil
}

// isRedundantRelease check if this release is a redundant one.
func isRedundantRelease(rlsName string, advDeploy *workloadv1beta1.AdvDeployment) bool {
	for _, podSet := range advDeploy.Spec.Topology.PodSets {
		if podSet.Name == rlsName {
			return false
		}
	}

	return true
}

// findRunningReleases find the release with its name from running release response.
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

func getChartInfo(podSet *workloadv1beta1.PodSet, advDeploy *workloadv1beta1.AdvDeployment) (specChartURLName string, specChartURLVersion string, specRawChart []byte) {

	if podSet.Chart != nil {
		if podSet.Chart.RawChart != nil {
			specRawChart = *podSet.Chart.RawChart
		}
		if podSet.Chart.ChartUrl != nil {
			specChartURLName = podSet.Chart.ChartUrl.Url
			specChartURLVersion = podSet.Chart.ChartUrl.ChartVersion
		}

		// if podset chart info is empty, use global chart info
		if specRawChart != nil || specChartURLName != "" || specChartURLVersion != "" {
			return specChartURLName, specChartURLVersion, specRawChart
		}
	}

	if advDeploy.Spec.PodSpec.Chart.RawChart != nil {
		specRawChart = *advDeploy.Spec.PodSpec.Chart.RawChart
	}

	if advDeploy.Spec.PodSpec.Chart.ChartUrl != nil {
		specChartURLName = advDeploy.Spec.PodSpec.Chart.ChartUrl.Url
		specChartURLVersion = advDeploy.Spec.PodSpec.Chart.ChartUrl.ChartVersion
	}

	return specChartURLName, specChartURLVersion, specRawChart
}
