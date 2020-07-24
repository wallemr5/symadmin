package advdeployment

import (
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
)

// isRedundantRelease check if this release is a redundant one.
func isRedundantRelease(rlsName string, advDeploy *workloadv1beta1.AdvDeployment) bool {
	for _, podSet := range advDeploy.Spec.Topology.PodSets {
		if podSet.Name == rlsName {
			return false
		}
	}

	return true
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
