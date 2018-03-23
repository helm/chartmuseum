package multitenant

import (
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
)

func (s *MultiTenantServer) Routes() []cm_router.Route {
	var routes []cm_router.Route

	serverInfoRoutes := []cm_router.Route{
		{"READ", "GET", "/", s.defaultHandler},
		{"SYSTEM", "GET", "/health", s.getHealthCheckHandler},
	}

	helmChartRepositoryRoutes := []cm_router.Route{
		{"READ", "GET", "/:repo/index.yaml", s.getIndexFileRequestHandler},
		{"READ", "GET", "/:repo/charts/:filename", s.getStorageObjectRequestHandler},
	}

	routes = append(routes, serverInfoRoutes...)
	routes = append(routes, helmChartRepositoryRoutes...)

	return routes
}
