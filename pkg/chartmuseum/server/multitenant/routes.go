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
	cm_auth "github.com/chartmuseum/auth"

	cm_router "helm.sh/chartmuseum/pkg/chartmuseum/router"
)

func (s *MultiTenantServer) Routes() []*cm_router.Route {
	var routes []*cm_router.Route

	serverInfoRoutes := []*cm_router.Route{
		{"GET", "/", s.getWelcomePageHandler, cm_auth.PullAction},
		{"GET", "/info", s.getInfoHandler, ""},
		{"GET", "/health", s.getHealthCheckHandler, ""},
	}

	artifactHubRoutes := []*cm_router.Route{
		{"GET", "/artifacthub-repo.yml", s.getArtifactHubFileRequestHandler, cm_auth.PullAction},
	}

	helmChartRepositoryRoutes := []*cm_router.Route{
		{"GET", "/:repo/index.yaml", s.getIndexFileRequestHandler, cm_auth.PullAction},
		{"GET", "/:repo/charts/:filename", s.getStorageObjectRequestHandler, cm_auth.PullAction},
	}

	chartManipulationRoutes := []*cm_router.Route{
		{"GET", "/api/:repo/charts", s.getAllChartsRequestHandler, cm_auth.PullAction},
		{"HEAD", "/api/:repo/charts/:name", s.headChartRequestHandler, cm_auth.PullAction},
		{"GET", "/api/:repo/charts/:name", s.getChartRequestHandler, cm_auth.PullAction},
		{"HEAD", "/api/:repo/charts/:name/:version", s.headChartVersionRequestHandler, cm_auth.PullAction},
		{"GET", "/api/:repo/charts/:name/:version", s.getChartVersionRequestHandler, cm_auth.PullAction},
		{"GET", "/api/:repo/charts/:name/:version/templates", s.getStorageObjectTemplateRequestHandler, cm_auth.PullAction},
		{"GET", "/api/:repo/charts/:name/:version/values", s.getStorageObjectValuesRequestHandler, cm_auth.PullAction},
		{"POST", "/api/:repo/charts", s.postRequestHandler, cm_auth.PushAction},
		{"POST", "/api/:repo/prov", s.postProvenanceFileRequestHandler, cm_auth.PushAction},
	}

	routes = append(routes, serverInfoRoutes...)
	routes = append(routes, helmChartRepositoryRoutes...)

	if len(s.ArtifactHubRepoID) != 0 {
		routes = append(routes, artifactHubRoutes...)
	}

	if s.WebTemplatePath != "" {
		routes = append(routes, &cm_router.Route{
			Method:  "GET",
			Path:    "/static",
			Handler: s.getStaticFilesHandler,
			Action:  cm_auth.PullAction,
		})
	}

	if s.APIEnabled {
		routes = append(routes, chartManipulationRoutes...)
	}

	if s.APIEnabled && !s.DisableDelete {
		routes = append(routes, &cm_router.Route{"DELETE", "/api/:repo/charts/:name/:version", s.deleteChartVersionRequestHandler, cm_auth.PushAction})
	}

	return routes
}
