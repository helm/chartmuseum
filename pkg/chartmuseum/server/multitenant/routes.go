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
		{Method: "GET", Path: "/", Handler: s.getWelcomePageHandler, Action: cm_auth.PullAction},
		{Method: "GET", Path: "/info", Handler: s.getInfoHandler, Action: ""},
		{Method: "GET", Path: "/health", Handler: s.getHealthCheckHandler, Action: ""},
	}

	artifactHubRoutes := []*cm_router.Route{
		{Method: "GET", Path: "/artifacthub-repo.yml", Handler: s.getArtifactHubFileRequestHandler, Action: cm_auth.PullAction},
	}

	helmChartRepositoryRoutes := []*cm_router.Route{
		{Method: "GET", Path: "/:repo/index.yaml", Handler: s.getIndexFileRequestHandler, Action: cm_auth.PullAction},
		{Method: "HEAD", Path: "/:repo/index.yaml", Handler: s.headIndexFileRequestHandler, Action: cm_auth.PullAction},
		{Method: "GET", Path: "/:repo/charts/:filename", Handler: s.getStorageObjectRequestHandler, Action: cm_auth.PullAction},
	}

	chartManipulationRoutes := []*cm_router.Route{
		{Method: "GET", Path: "/api/:repo/charts", Handler: s.getAllChartsRequestHandler, Action: cm_auth.PullAction},
		{Method: "HEAD", Path: "/api/:repo/charts/:name", Handler: s.headChartRequestHandler, Action: cm_auth.PullAction},
		{Method: "GET", Path: "/api/:repo/charts/:name", Handler: s.getChartRequestHandler, Action: cm_auth.PullAction},
		{Method: "HEAD", Path: "/api/:repo/charts/:name/:version", Handler: s.headChartVersionRequestHandler, Action: cm_auth.PullAction},
		{Method: "GET", Path: "/api/:repo/charts/:name/:version", Handler: s.getChartVersionRequestHandler, Action: cm_auth.PullAction},
		{Method: "GET", Path: "/api/:repo/charts/:name/:version/templates", Handler: s.getStorageObjectTemplateRequestHandler, Action: cm_auth.PullAction},
		{Method: "GET", Path: "/api/:repo/charts/:name/:version/values", Handler: s.getStorageObjectValuesRequestHandler, Action: cm_auth.PullAction},
		{Method: "POST", Path: "/api/:repo/charts", Handler: s.postRequestHandler, Action: cm_auth.PushAction},
		{Method: "POST", Path: "/api/:repo/prov", Handler: s.postProvenanceFileRequestHandler, Action: cm_auth.PushAction},
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
		routes = append(routes, &cm_router.Route{Method: "DELETE", Path: "/api/:repo/charts/:name/:version", Handler: s.deleteChartVersionRequestHandler, Action: cm_auth.PushAction})
	}

	return routes
}
