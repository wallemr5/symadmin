package v1

import (
	"gitlab.dmall.com/arch/sym-admin/pkg/router"
)

// Routes ...
func (m *Manager) Routes() []*router.Route {
	var routes []*router.Route

	apiRoutes := []*router.Route{
		{
			Method:  "GET",
			Path:    "/api/cluster/:name",
			Handler: m.GetClusters,
			Desc:    GetClusterDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/app/:appName/resource",
			Handler: m.GetClusterResource,
			Desc:    GetClusterResourceDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/appPods/labels",
			Handler: m.GetPodByLabels,
			Desc:    GetPodByLabelsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/ip/:podIP",
			Handler: m.GetPodByIP,
			Desc:    GetPodByLabelsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/appPod/:appName/helm",
			Handler: m.GetHelmReleases,
			Desc:    GetHelmReleasesDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/helm/:releaseName",
			Handler: m.GetHelmReleaseInfo,
			Desc:    GetHelmReleaseInfoDesc,
		},
		{
			Method:  "POST",
			Path:    "/api/linthelm",
			Handler: m.LintHelmTemplate,
			Desc:    GetHelmReleaseInfoDesc,
		},
		{
			Method:  "POST",
			Path:    "/api/cluster/:name/namespace/:namespace/app/:appName/restart",
			Handler: m.DeletePodByGroup,
			Desc:    DeletePodByGroupDesc,
		},
		{
			Method:  "DELETE",
			Path:    "/api/cluster/:name/namespace/:namespace/pod/:podName",
			Handler: m.DeletePodByName,
			Desc:    DeletePodByNameDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/pod/:podName",
			Handler: m.GetPodByName,
			Desc:    GetPodByNameDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/endpoints/:appName",
			Handler: m.GetEndpoints,
			Desc:    GetEndpointsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/terminal",
			Handler: m.GetTerminal,
			Desc:    GetTerminalDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/exec",
			Handler: m.ExecOnceWithHTTP,
			Desc:    ExecOnceWithHTTPDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/services/:appName",
			Handler: m.GetServices,
			Desc:    GetServicesDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/service/:svcName",
			Handler: m.GetServiceInfo,
			Desc:    GetServiceInfoDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/deployments/:appName",
			Handler: m.GetDeployments,
			Desc:    GetDeploymentsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/deployment/:deployName",
			Handler: m.GetDeploymentInfo,
			Desc:    GetDeploymentInfoDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/deployments/stat",
			Handler: m.GetDeploymentsStat,
			Desc:    GetDeploymentsStatDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/pods/:podName/event",
			Handler: m.GetPodEvent,
			Desc:    GetPodEventDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/events/warning",
			Handler: m.GetWarningEvents,
			Desc:    GetWarningEventsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/pod/logfiles",
			Handler: m.GetFiles,
			Desc:    GetFilesDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/pods/:podName/logs",
			Handler: m.HandleLogs,
			Desc:    HandleLogsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/namespace/:namespace/pods/:podName/logs/file",
			Handler: m.HandleFileLogs,
			Desc:    HandleFileLogsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/offlineWorkloadDeploy",
			Handler: m.HandleOfflineWorkloadDeploy,
		},
		{
			Method:  "GET",
			Path:    "/api/offlinePodAppList/all",
			Handler: m.GetAllOfflineApp,
		},
		{
			Method:  "GET",
			Path:    "/api/namespace/:namespace/appname/:appname/offlinepodlist",
			Handler: m.GetOfflinePods,
		},
		{
			Method:  "GET",
			Path:    "/api/cluster/:name/offlineWorkloadPod/terminal",
			Handler: m.GetOfflineLogTerminal,
		},
		{
			Method:  "GET",
			Path:    "/api/lintLocalTemplate/",
			Handler: m.LintLocalTemplate,
		},
	}

	routes = append(routes, apiRoutes...)
	return routes
}
