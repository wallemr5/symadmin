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
			Path:    "/v2/api/cluster/:name",
			Handler: m.GetClusters,
			Desc:    GetClusterDesc,
		},
	}

	routes = append(routes, apiRoutes...)
	return routes
}
