/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package multitenant

import (
	cm_router "github.com/helm/chartmuseum/pkg/chartmuseum/router"
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
