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
			Path:    "/api/v2/cluster/:name",
			Handler: m.GetClusters,
			Desc:    GetClusterDesc,
		},
		{
			Method:  "GET",
			Path:    "/api/v2/cluster/:clusterCode/appPods/labels",
			Handler: m.GetPodByLabels,
			Desc:    GetPodByLabelsDesc,
		},
	}

	routes = append(routes, apiRoutes...)
	return routes
}
