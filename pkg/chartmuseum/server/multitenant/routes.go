package multitenant

import (
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
)

func (s *MultiTenantServer) Routes() []*cm_router.Route {
	var routes []*cm_router.Route

	serverInfoRoutes := []*cm_router.Route{
		{"GET", "/", s.getWelcomePageHandler, cm_router.RepoPullAction},
		{"GET", "/health", s.getHealthCheckHandler, cm_router.SystemInfoAction},
	}

	helmChartRepositoryRoutes := []*cm_router.Route{
		{"GET", "/:repo/index.yaml", s.getIndexFileRequestHandler, cm_router.RepoPullAction},
		{"GET", "/:repo/charts/:filename", s.getStorageObjectRequestHandler, cm_router.RepoPullAction},
	}

	chartManipulationRoutes := []*cm_router.Route{
		{"GET", "/api/:repo/charts", s.getAllChartsRequestHandler, cm_router.RepoPullAction},
		{"GET", "/api/:repo/charts/:name", s.getChartRequestHandler, cm_router.RepoPullAction},
		{"GET", "/api/:repo/charts/:name/:version", s.getChartVersionRequestHandler, cm_router.RepoPullAction},
		{"POST", "/api/:repo/charts", s.postRequestHandler, cm_router.RepoPushAction},
		{"POST", "/api/:repo/prov", s.postProvenanceFileRequestHandler, cm_router.RepoPushAction},
		{"DELETE", "/api/:repo/charts/:name/:version", s.deleteChartVersionRequestHandler, cm_router.RepoPushAction},
	}

	routes = append(routes, serverInfoRoutes...)
	routes = append(routes, helmChartRepositoryRoutes...)

	if s.APIEnabled {
		routes = append(routes, chartManipulationRoutes...)
	}

	return routes
}
