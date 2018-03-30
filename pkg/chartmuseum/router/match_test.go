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
		{"GET", "/", handlers[0]},
		{"GET", "/health", handlers[1]},
		{"GET", "/:repo/index.yaml", handlers[2]},
		{"GET", "/:repo/charts/:filename", handlers[3]},
		{"GET", "/api/:repo/charts", handlers[4]},
		{"GET", "/api/:repo/charts/:name", handlers[5]},
		{"GET", "/api/:repo/charts/:name/:version", handlers[6]},
		{"POST", "/api/:repo/charts", handlers[7]},
		{"POST", "/api/:repo/prov", handlers[8]},
		{"DELETE", "/api/:repo/charts/:name/:version", handlers[9]},
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
			handle, params := match(routes, "GET", pathutil.Join("/", contextPath), contextPath, 0)
			suite.NotNil(handle)
			suite.Nil(params)
			if handle != nil {
				handle(c)
			}
			val, exists := c.Get("index")
			suite.True(exists)
			suite.Equal(0, val)

			// GET /health
			handle, params = match(routes, "GET", pathutil.Join("/", contextPath, "health"), contextPath, 0)
			suite.NotNil(handle)
			suite.Nil(params)
			if handle != nil {
				handle(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(1, val)

			// GET /index.yaml
			route := pathutil.Join("/", contextPath, repo, "index.yaml")
			handle, params = match(routes, "GET", route, contextPath, depth)
			suite.NotNil(handle)
			if handle != nil {
				handle(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(2, val)
			suite.Equal([]gin.Param{{"repo", repo}}, params)

			// GET /charts/mychart-0.1.0.tgz
			route = pathutil.Join("/", contextPath, repo, "charts/mychart-0.1.0.tgz")
			handle, params = match(routes, "GET", route, contextPath, depth)
			suite.NotNil(handle)
			if handle != nil {
				handle(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(3, val)
			suite.Equal([]gin.Param{{"filename", "mychart-0.1.0.tgz"}, {"repo", repo}}, params)

			// GET /api/charts
			route = pathutil.Join("/", contextPath, "api", repo, "charts")
			handle, params = match(routes, "GET", route, contextPath, depth)
			suite.NotNil(handle)
			if handle != nil {
				handle(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(4, val)
			suite.Equal([]gin.Param{{"repo", repo}}, params)

			// GET /api/charts/mychart
			route = pathutil.Join("/", contextPath, "api", repo, "charts/mychart")
			handle, params = match(routes, "GET", route, contextPath, depth)
			suite.NotNil(handle)
			if handle != nil {
				handle(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(5, val)
			suite.Equal([]gin.Param{{"name", "mychart"}, {"repo", repo}}, params)

			// GET /api/charts/mychart/0.1.0
			route = pathutil.Join("/", contextPath, "api", repo, "charts/mychart/0.1.0")
			handle, params = match(routes, "GET", route, contextPath, depth)
			suite.NotNil(handle)
			if handle != nil {
				handle(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(6, val)
			suite.Equal([]gin.Param{{"name", "mychart"}, {"version", "0.1.0"}, {"repo", repo}}, params)

			// POST /api/charts
			route = pathutil.Join("/", contextPath, "api", repo, "charts")
			handle, params = match(routes, "POST", route, contextPath, depth)
			suite.NotNil(handle)
			if handle != nil {
				handle(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(7, val)
			suite.Equal([]gin.Param{{"repo", repo}}, params)

			// POST /api/prov
			route = pathutil.Join("/", contextPath, "api", repo, "prov")
			handle, params = match(routes, "POST", route, contextPath, depth)
			suite.NotNil(handle)
			if handle != nil {
				handle(c)
			}
			val, exists = c.Get("index")
			suite.True(exists)
			suite.Equal(8, val)
			suite.Equal([]gin.Param{{"repo", repo}}, params)

			// DELETE /api/charts/mychart/0.1.0
			route = pathutil.Join("/", contextPath, "api", repo, "charts/mychart/0.1.0")
			handle, params = match(routes, "DELETE", route, contextPath, depth)
			suite.NotNil(handle)
			if handle != nil {
				handle(c)
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
