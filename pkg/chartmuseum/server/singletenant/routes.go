package singletenant

import (
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
)

func (s *SingleTenantServer) Routes() []*cm_router.Route {
	var routes []*cm_router.Route

	serverInfoRoutes := []*cm_router.Route{
		{"GET", "/", s.getWelcomePageHandler},
		{"GET", "/health", s.getHealthCheckHandler},
	}

	helmChartRepositoryRoutes := []*cm_router.Route{
		{"GET", "/:repo/index.yaml", s.getIndexFileRequestHandler},
		{"GET", "/:repo/charts/:filename", s.getStorageObjectRequestHandler},
	}

	chartManipulationRoutes := []*cm_router.Route{
		{"GET", "/api/:repo/charts", s.getAllChartsRequestHandler},
		{"GET", "/api/:repo/charts/:name", s.getChartRequestHandler},
		{"GET", "/api/:repo/charts/:name/:version", s.getChartVersionRequestHandler},
		{"POST", "/api/:repo/charts", s.postRequestHandler},
		{"POST", "/api/:repo/prov", s.postProvenanceFileRequestHandler},
		{"DELETE", "/api/:repo/charts/:name/:version", s.deleteChartVersionRequestHandler},
	}

	routes = append(routes, serverInfoRoutes...)
	routes = append(routes, helmChartRepositoryRoutes...)

	if s.APIEnabled {
		routes = append(routes, chartManipulationRoutes...)
	}

	return routes
}
