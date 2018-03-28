package singletenant

import (
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
)

func (s *SingleTenantServer) Routes() []*cm_router.Route {
	var routes []*cm_router.Route

	serverInfoRoutes := []*cm_router.Route{
		{"READ", "GET", "/", s.getWelcomePageHandler},
		{"SYSTEM", "GET", "/health", s.getHealthCheckHandler},
	}

	helmChartRepositoryRoutes := []*cm_router.Route{
		{"READ", "GET", "/:repo/index.yaml", s.getIndexFileRequestHandler},
		{"READ", "GET", "/:repo/charts/:filename", s.getStorageObjectRequestHandler},
	}

	chartManipulationRoutes := []*cm_router.Route{
		{"READ", "GET", "/api/:repo/charts", s.getAllChartsRequestHandler},
		{"READ", "GET", "/api/:repo/charts/:name", s.getChartRequestHandler},
		{"READ", "GET", "/api/:repo/charts/:name/:version", s.getChartVersionRequestHandler},
		{"WRITE", "POST", "/api/:repo/charts", s.postRequestHandler},
		{"WRITE", "POST", "/api/:repo/prov", s.postProvenanceFileRequestHandler},
		{"WRITE", "DELETE", "/api/:repo/charts/:name/:version", s.deleteChartVersionRequestHandler},
	}

	routes = append(routes, serverInfoRoutes...)
	routes = append(routes, helmChartRepositoryRoutes...)

	if s.APIEnabled {
		routes = append(routes, chartManipulationRoutes...)
	}

	return routes
}
