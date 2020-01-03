package advdeployment

import (
	"context"

	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	hapirelease "k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/klog"
)

func (r *AdvDeploymentReconciler) CleanReleasesByName(advDeploy *workloadv1beta1.AdvDeployment) error {
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

	for _, rls := range response.Releases {
		err := helmv2.DeleteRelease(rls.Name, hClient)
		if err != nil {
			klog.Errorf("DeleteRelease name: %s, err:%+v", rls.Name, err)
			return err
		}
	}

	return nil
}

func (r *AdvDeploymentReconciler) ApplyPodSetReleases(ctx context.Context, advDeploy *workloadv1beta1.AdvDeployment) error {
	hClient, err := helmv2.NewClientFromConfig(r.Cfg, r.KubeCli, "")
	if err != nil {
		klog.Errorf("New helm Clinet err:%+v", err)
		return err
	}
	defer hClient.Close()

	response, err := helmv2.ListReleases(labels.MakeHelmReleaseFilter(advDeploy.Name), hClient)
	if err != nil {
		klog.Errorf("no find helm releases by name: %s, err: %v", advDeploy.Name, err)
		return err
	}

	var unUseReleasesName []string
	if response != nil {
		for _, r := range response.Releases {
			if !isFindPodSetByName(r.Name, advDeploy) {
				unUseReleasesName = append(unUseReleasesName, r.Name)
			}

			if r.Info.Status.GetCode() != hapirelease.Status_DEPLOYED {
				klog.Infof("rlsName: %s deploy failed, Description:%s", r.Name, r.Info.Description)
				err := helmv2.DeleteRelease(r.Name, hClient)
				if err != nil {
					klog.Errorf("DeleteRelease UNDEPLOYED rls name: %s, err:%+v", r.Name, err)
					return err
				}
			}
		}
	}

	var (
		chartUrlName    string
		chartUrlVersion string
		RawChart        []byte
		// rls    *hapirelease.Release
		rlsErr error
	)

	if advDeploy.Spec.PodSpec.Chart.RawChart != nil {
		RawChart = *advDeploy.Spec.PodSpec.Chart.RawChart
	}

	if advDeploy.Spec.PodSpec.Chart.ChartUrl != nil {
		chartUrlName = advDeploy.Spec.PodSpec.Chart.ChartUrl.Url
		chartUrlVersion = advDeploy.Spec.PodSpec.Chart.ChartUrl.ChartVersion
	}
	for _, podSet := range advDeploy.Spec.Topology.PodSets {
		// rls = nil
		rlsErr = nil

		_, rlsErr = helmv2.ApplyRelease(podSet.Name, chartUrlName, chartUrlVersion, RawChart,
			hClient, r.HelmEnv.Helmv2env, advDeploy.Namespace, getReleasesByName(podSet.Name, response), []byte(podSet.RawValues))
		if rlsErr != nil {
			klog.Errorf("podSet name: %s, ApplyRelease err: %v", podSet.Name, err)
			return rlsErr
		}
	}

	for _, rlsName := range unUseReleasesName {
		err := helmv2.DeleteRelease(rlsName, hClient)
		if err != nil {
			klog.Errorf("DeleteRelease name: %s, err:%+v", rlsName, err)
			return err
		}
	}

	return nil
}

func isFindPodSetByName(name string, advDeploy *workloadv1beta1.AdvDeployment) bool {
	for _, podSet := range advDeploy.Spec.Topology.PodSets {
		if podSet.Name == name {
			return true
		}
	}

	return false
}

func getReleasesByName(name string, rlsList *rls.ListReleasesResponse) *hapirelease.Release {
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
