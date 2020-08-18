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

package router

import (
	"net/http/httptest"
	pathutil "path"
	"testing"
	"strings"
	"sort"

	cm_auth "github.com/chartmuseum/auth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type MatchTestSuite struct {
	suite.Suite
}

type GinParamList struct {
	params []gin.Param
}

func (l GinParamList) Len() int {
	return len(l.params)
}

func (l GinParamList) Less(a int, b int) bool {
	return strings.Compare(l.params[a].Key, l.params[b].Key) < 0
}

func (l GinParamList) Swap(a int, b int) {
	l.params[a], l.params[b] = l.params[b], l.params[a]
}

func sortParams(params []gin.Param) []gin.Param {
	l := GinParamList{params: params}
	sort.Sort(l)
	return l.params
}

func (suite *MatchTestSuite) TestMatch() {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	handlers := []gin.HandlerFunc{}

	for i := 0; i <= 10; i++ {
		{
			j := i
			handlers = append(handlers, func(c *gin.Context) {
				c.Set("index", j)
			})
		}
	}

	routes := []*Route{
		{"GET", "/", handlers[0], cm_auth.PullAction},
		{"GET", "/health", handlers[1], ""},
		{"GET", "/:repo/index.yaml", handlers[2], cm_auth.PullAction},
		{"GET", "/:repo/charts/:filename", handlers[3], cm_auth.PullAction},
		{"GET", "/api/:repo/charts", handlers[4], cm_auth.PullAction},
		{"GET", "/api/:repo/charts/:name", handlers[5], cm_auth.PullAction},
		{"GET", "/api/:repo/charts/:name/:version", handlers[6], cm_auth.PullAction},
		{"POST", "/api/:repo/charts", handlers[7], cm_auth.PushAction},
		{"POST", "/api/:repo/prov", handlers[8], cm_auth.PushAction},
		{"DELETE", "/api/:repo/charts/:name/:version", handlers[9], cm_auth.PushAction},
		{"GET", "/api/:repofragment/repos", handlers[10], cm_auth.PullAction},
	}

	for depth, repo := range []string{"", "myrepo", "myorg/myrepo", "myorg/myteam/myrepo"} {
		for _, contextPath := range []string{"", "/x", "/x/y", "/x/y/z"} {
			// GET /
			r := pathutil.Join("/", contextPath)
			route, params := match(routes, "GET", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic := match(routes, "GET", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			suite.Nil(params)
			if route != nil {
				route.Handler(c)
			}
			val, exists := c.Get("index")
			suite.True(exists)
			suite.Equal(0, val)

			// GET /health
			r = pathutil.Join("/", contextPath, "health")
			route, params = match(routes, "GET", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			suite.Nil(params)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(1, val)

			// GET /index.yaml
			r = pathutil.Join("/", contextPath, repo, "index.yaml")
			route, params = match(routes, "GET", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(2, val)
			suite.Equal(sortParams([]gin.Param{{"repo", repo}}), sortParams(params))

			// GET /charts/mychart-0.1.0.tgz
			r = pathutil.Join("/", contextPath, repo, "charts/mychart-0.1.0.tgz")
			route, params = match(routes, "GET", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(3, val)
			suite.Equal(sortParams([]gin.Param{{"filename", "mychart-0.1.0.tgz"}, {"repo", repo}}), sortParams(params))

			// GET /api/charts
			r = pathutil.Join("/", contextPath, "api", repo, "charts")
			route, params = match(routes, "GET", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(4, val)
			suite.Equal(sortParams([]gin.Param{{"repo", repo}}), sortParams(params))

			// GET /api/charts/mychart
			r = pathutil.Join("/", contextPath, "api", repo, "charts/mychart")
			route, params = match(routes, "GET", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(5, val)
			suite.Equal(sortParams([]gin.Param{{"name", "mychart"}, {"repo", repo}}), sortParams(params))

			// GET /api/charts/mychart/0.1.0
			r = pathutil.Join("/", contextPath, "api", repo, "charts/mychart/0.1.0")
			route, params = match(routes, "GET", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(6, val)
			suite.Equal(sortParams([]gin.Param{{"name", "mychart"}, {"version", "0.1.0"}, {"repo", repo}}), sortParams(params))

			// POST /api/charts
			r = pathutil.Join("/", contextPath, "api", repo, "charts")
			route, params = match(routes, "POST", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "POST", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(7, val)
			suite.Equal(sortParams([]gin.Param{{"repo", repo}}), sortParams(params))

			// POST /api/prov
			r = pathutil.Join("/", contextPath, "api", repo, "prov")
			route, params = match(routes, "POST", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "POST", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(8, val)
			suite.Equal(sortParams([]gin.Param{{"repo", repo}}), sortParams(params))

			// DELETE /api/charts/mychart/0.1.0
			r = pathutil.Join("/", contextPath, "api", repo, "charts/mychart/0.1.0")
			route, params = match(routes, "DELETE", r, contextPath, depth, false)
			routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "DELETE", r, contextPath, 0, true)
			suite.Equal(route, routeWithDepthDynamic)
			suite.Equal(params, paramsWithDepthDynamic)

			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(9, val)
			suite.Equal(sortParams([]gin.Param{{"name", "mychart"}, {"version", "0.1.0"}, {"repo", repo}}), sortParams(params))

			// GET /api/repos
			for fragmentDepth, fragment := range []string{"", "myrepo", "myorg/myrepo", "myorg/myteam/myrepo"} {
				r = pathutil.Join("/", contextPath, "api", fragment, "repos")
				route, params = match(routes, "GET", r, contextPath, depth, false)
				routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, contextPath, 0, true)
				suite.NotNil(routeWithDepthDynamic)

				if fragmentDepth >= depth {
					suite.Nil(route)
				} else {
					suite.Equal(route, routeWithDepthDynamic)
					suite.Equal(params, paramsWithDepthDynamic)
					suite.NotNil(route)
					if route != nil {
						route.Handler(c)
					}
					val, exists = c.Get("index")
					suite.True(exists)
					suite.Equal(10, val)
					suite.Equal(sortParams([]gin.Param{{"repofragment", fragment}}), sortParams(params))
				}
			}
		}
	}

	// Test route repos named "api*"
	r := "/apix/index.yaml"
	route, params := match(routes, "GET", r, "", 1, false)
	routeWithDepthDynamic, paramsWithDepthDynamic := match(routes, "GET", r, "", 0, true)
	suite.Equal(route, routeWithDepthDynamic)
	suite.Equal(params, paramsWithDepthDynamic)

	suite.NotNil(route)
	if route != nil {
		route.Handler(c)
	}
	val, exists := c.Get("index")
	suite.True(exists)
	suite.Equal(2, val)
	suite.Equal(sortParams([]gin.Param{{"repo", "apix"}}), sortParams(params))

	r = "/apix/charts/mychart-0.1.0.tgz"
	route, params = match(routes, "GET", r, "", 1, false)
	routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, "", 0, true)
	suite.Equal(route, routeWithDepthDynamic)
	suite.Equal(params, paramsWithDepthDynamic)

	suite.NotNil(route)
	if route != nil {
		route.Handler(c)
	}
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(3, val)
	suite.Equal(sortParams([]gin.Param{{"filename", "mychart-0.1.0.tgz"}, {"repo", "apix"}}), sortParams(params))

	// Test route repos named just "api"
	r = "/api/index.yaml"
	route, params = match(routes, "GET", r, "", 1, false)
	routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, "", 0, true)
	suite.Equal(route, routeWithDepthDynamic)
	suite.Equal(params, paramsWithDepthDynamic)

	suite.NotNil(route)
	if route != nil {
		route.Handler(c)
	}
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(2, val)
	suite.Equal(sortParams([]gin.Param{{"repo", "api"}}), sortParams(params))

	r = "/api/charts/mychart-0.1.0.tgz"
	route, params = match(routes, "GET", r, "", 1, false)
	routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, "", 0, true)
	suite.Equal(route, routeWithDepthDynamic)
	suite.Equal(params, paramsWithDepthDynamic)

	suite.NotNil(route)
	if route != nil {
		route.Handler(c)
	}
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(3, val)
	suite.Equal(sortParams([]gin.Param{{"filename", "mychart-0.1.0.tgz"}, {"repo", "api"}}), sortParams(params))

	// just "api" as repo name, depth=2
	r = "/api/xyz/index.yaml"
	route, params = match(routes, "GET", r, "", 2, false)
	routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, "", 0, true)
	suite.Equal(route, routeWithDepthDynamic)
	suite.Equal(params, paramsWithDepthDynamic)

	suite.NotNil(route)
	if route != nil {
		route.Handler(c)
	}
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(2, val)
	suite.Equal(sortParams([]gin.Param{{"repo", "api/xyz"}}), sortParams(params))

	r = "/api/xyz/charts/mychart-0.1.0.tgz"
	route, params = match(routes, "GET", r, "", 2, false)
	routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, "", 0, true)
	suite.Equal(route, routeWithDepthDynamic)
	suite.Equal(params, paramsWithDepthDynamic)

	suite.NotNil(route)
	if route != nil {
		route.Handler(c)
	}
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(3, val)
	suite.Equal(sortParams([]gin.Param{{"filename", "mychart-0.1.0.tgz"}, {"repo", "api/xyz"}}), sortParams(params))

	// Test route repos named "health"
	r = "/health/index.yaml"
	route, params = match(routes, "GET", r, "", 1, false)
	routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, "", 0, true)
	suite.Equal(route, routeWithDepthDynamic)
	suite.Equal(params, paramsWithDepthDynamic)

	suite.NotNil(route)
	if route != nil {
		route.Handler(c)
	}
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(2, val)
	suite.Equal(sortParams([]gin.Param{{"repo", "health"}}), sortParams(params))

	r = "/health/charts/mychart-0.1.0.tgz"
	route, params = match(routes, "GET", r, "", 1, false)
	routeWithDepthDynamic, paramsWithDepthDynamic = match(routes, "GET", r, "", 0, true)
	suite.Equal(route, routeWithDepthDynamic)
	suite.Equal(params, paramsWithDepthDynamic)

	suite.NotNil(route)
	if route != nil {
		route.Handler(c)
	}
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(3, val)
	suite.Equal(sortParams([]gin.Param{{"filename", "mychart-0.1.0.tgz"}, {"repo", "health"}}), sortParams(params))
}

func TestMatchTestSuite(t *testing.T) {
	suite.Run(t, new(MatchTestSuite))
}
