package router

import (
	"net/http/httptest"
	pathutil "path"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type MatchTestSuite struct {
	suite.Suite
}

func (suite *MatchTestSuite) TestMatch() {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	handlers := []gin.HandlerFunc{}

	for i := 0; i <= 9; i++ {
		{
			j := i
			handlers = append(handlers, func(c *gin.Context) {
				c.Set("index", j)
			})
		}
	}

	routes := []*Route{
		{"GET", "/", handlers[0], RepoPullAction},
		{"GET", "/health", handlers[1], SystemInfoAction},
		{"GET", "/:repo/index.yaml", handlers[2], RepoPullAction},
		{"GET", "/:repo/charts/:filename", handlers[3], RepoPullAction},
		{"GET", "/api/:repo/charts", handlers[4], RepoPullAction},
		{"GET", "/api/:repo/charts/:name", handlers[5], RepoPullAction},
		{"GET", "/api/:repo/charts/:name/:version", handlers[6], RepoPullAction},
		{"POST", "/api/:repo/charts", handlers[7], RepoPushAction},
		{"POST", "/api/:repo/prov", handlers[8], RepoPushAction},
		{"DELETE", "/api/:repo/charts/:name/:version", handlers[9], RepoPushAction},
	}

	for depth := 0; depth <= 3; depth++ {
		var repo string

		switch {
		case depth == 1:
			repo = "myrepo"
		case depth == 2:
			repo = "myorg/myrepo"
		case depth == 3:
			repo = "myorg/myteam/myrepo"
		}

		for _, contextPath := range []string{"", "/x", "/x/y", "/x/y/z"} {

			// GET /
			r := pathutil.Join("/", contextPath)
			route, params := match(routes, "GET", r, contextPath, 0)
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
			route, params = match(routes, "GET", r, contextPath, 0)
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
			route, params = match(routes, "GET", r, contextPath, depth)
			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(2, val)
			suite.Equal([]gin.Param{{"repo", repo}}, params)

			// GET /charts/mychart-0.1.0.tgz
			r = pathutil.Join("/", contextPath, repo, "charts/mychart-0.1.0.tgz")
			route, params = match(routes, "GET", r, contextPath, depth)
			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(3, val)
			suite.Equal([]gin.Param{{"filename", "mychart-0.1.0.tgz"}, {"repo", repo}}, params)

			// GET /api/charts
			r = pathutil.Join("/", contextPath, "api", repo, "charts")
			route, params = match(routes, "GET", r, contextPath, depth)
			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(4, val)
			suite.Equal([]gin.Param{{"repo", repo}}, params)

			// GET /api/charts/mychart
			r = pathutil.Join("/", contextPath, "api", repo, "charts/mychart")
			route, params = match(routes, "GET", r, contextPath, depth)
			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(5, val)
			suite.Equal([]gin.Param{{"name", "mychart"}, {"repo", repo}}, params)

			// GET /api/charts/mychart/0.1.0
			r = pathutil.Join("/", contextPath, "api", repo, "charts/mychart/0.1.0")
			route, params = match(routes, "GET", r, contextPath, depth)
			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(6, val)
			suite.Equal([]gin.Param{{"name", "mychart"}, {"version", "0.1.0"}, {"repo", repo}}, params)

			// POST /api/charts
			r = pathutil.Join("/", contextPath, "api", repo, "charts")
			route, params = match(routes, "POST", r, contextPath, depth)
			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(7, val)
			suite.Equal([]gin.Param{{"repo", repo}}, params)

			// POST /api/prov
			r = pathutil.Join("/", contextPath, "api", repo, "prov")
			route, params = match(routes, "POST", r, contextPath, depth)
			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(8, val)
			suite.Equal([]gin.Param{{"repo", repo}}, params)

			// DELETE /api/charts/mychart/0.1.0
			r = pathutil.Join("/", contextPath, "api", repo, "charts/mychart/0.1.0")
			route, params = match(routes, "DELETE", r, contextPath, depth)
			suite.NotNil(route)
			if route != nil {
				route.Handler(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(9, val)
			suite.Equal([]gin.Param{{"name", "mychart"}, {"version", "0.1.0"}, {"repo", repo}}, params)
		}
	}
}

func TestMatchTestSuite(t *testing.T) {
	suite.Run(t, new(MatchTestSuite))
}
