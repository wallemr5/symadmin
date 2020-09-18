package v2

import (
	"gitlab.dmall.com/arch/sym-admin/pkg/router"
)

// Routes ...
func (m *Manager) Routes() []*router.Route {
	var routes []*router.Route

	apiRoutes := []*router.Route{
		{
			Method:  "GET",
			Path:    "/api/v2/cluster/:clusterCode",
			Handler: m.GetClusters,
			Desc:    GetClusterDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/v2/cluster/:clusterCode/pods",
			Handler: m.GetPodByLabels,
			Desc:    GetPodByLabelsDesc,
		},
		{
			Method:  "DELETE",
			Path:    "/api/v2/cluster/:clusterCode/namespace/:namespace/pod/:podName",
			Handler: m.DeletePodByName,
			Desc:    DeletePodByNameDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/v2/cluster/:clusterCode/namespace/:namespace/pods/:podName/logs",
			Handler: m.HandleLogs,
			Desc:    HandleLogsDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/v2/cluster/:clusterCode/namespace/:namespace/pods/:podName/event",
			Handler: m.GetPodEvent,
			Desc:    GetPodEventDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/v2/cluster/:clusterCode/namespace/:namespace/pods/:podName/tail",
			Handler: m.TailFile,
			Desc:    TailFileDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/v2/cluster/:clusterCode/namespace/:namespace/pods/:podName/files",
			Handler: m.ListFiles,
			Desc:    ListFileDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/v2/cluster/:clusterCode/app/group/version",
			Handler: m.GetAppGroupVersion,
			Desc:    GetAppGroupVersionDesc,
		},
	}

	routes = append(routes, apiRoutes...)
	return routes
}
