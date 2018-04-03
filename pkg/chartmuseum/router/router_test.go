package router

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/suite"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"

	"github.com/gin-gonic/gin"
	"net/http/httptest"
)

type RouterTestSuite struct {
	suite.Suite
}

func (suite *RouterTestSuite) TestRouterHandleContext() {
	log, err := cm_logger.NewLogger(cm_logger.LoggerOptions{
		Debug:   true,
		LogJSON: true,
	})
	suite.Nil(err, "no error creating logger")

	// Trigger 404s and 500s
	routerMetricsEnabled := NewRouter(RouterOptions{
		Logger:        log,
		EnableMetrics: true,
	})
	testContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/", nil)
	routerMetricsEnabled.HandleContext(testContext)
	suite.Equal(404, testContext.Writer.Status())
	prefixed500Path := "/health"
	routerMetricsEnabled.GET(prefixed500Path, func(c *gin.Context) {
		c.Data(500, "text/html", []byte("500"))
	})
	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/health", nil)
	routerMetricsEnabled.HandleContext(testContext)
	suite.Equal(500, testContext.Writer.Status())

	testRoutes := []*Route{
		{"GET", "/", func(c *gin.Context) {
			c.Data(200, "text/html", []byte("200"))
		}, RepoPullAction},
		{"GET", "/health", func(c *gin.Context) {
			c.Data(200, "text/html", []byte("200"))
		}, SystemInfoAction},
		{"GET", "/:repo/whatsmyrepo", func(c *gin.Context) {
			c.Data(200, "text/html", []byte(c.GetString("repo")))
		}, RepoPullAction},
		{"GET", "/api/:repo/whatsmyrepo", func(c *gin.Context) {
			c.Data(200, "text/html", []byte(c.GetString("repo")))
		}, RepoPullAction},
		{"POST", "/api/:repo/writetorepo", func(c *gin.Context) {
			c.Data(200, "text/html", []byte(c.GetString("repo")))
		}, RepoPushAction},
		{"GET", "/api/:repo/systemstats", func(c *gin.Context) {
			c.Data(200, "text/html", []byte(c.GetString("repo")))
		}, RepoPullAction},
	}

	// Test route transformations
	router := NewRouter(RouterOptions{
		Logger: log,
		Depth:  3,
	})
	router.SetRoutes(testRoutes)

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/", nil)
	router.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/health", nil)
	router.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/x/y/z/whatsmyrepo", nil)
	router.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())
	suite.Equal("x/y/z", testContext.Param("repo"))

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/api/x/y/z/whatsmyrepo", nil)
	router.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())
	suite.Equal("x/y/z", testContext.Param("repo"))

	// Test custom context path
	customContextPathRouter := NewRouter(RouterOptions{
		Logger:      log,
		Depth:       3,
		ContextPath: "/my/crazy/path",
	})
	customContextPathRouter.SetRoutes(testRoutes)

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/index.yaml", nil)
	customContextPathRouter.HandleContext(testContext)
	suite.Equal(404, testContext.Writer.Status())

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/my/crazy/path", nil)
	customContextPathRouter.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/my/crazy/path/health", nil)
	customContextPathRouter.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/my/crazy/path/x/y/z/whatsmyrepo", nil)
	customContextPathRouter.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())
	suite.Equal("x/y/z", testContext.Param("repo"))

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/my/crazy/path/api/x/y/z/whatsmyrepo", nil)
	customContextPathRouter.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())
	suite.Equal("x/y/z", testContext.Param("repo"))

	// Test basic auth
	basicAuthRouter := NewRouter(RouterOptions{
		Logger:   log,
		Depth:    0,
		Username: "testuser",
		Password: "testpass",
	})
	basicAuthRouter.SetRoutes(testRoutes)

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/health", nil)
	basicAuthRouter.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/", nil)
	basicAuthRouter.HandleContext(testContext)
	suite.Equal(401, testContext.Writer.Status())

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/", nil)
	testContext.Request.SetBasicAuth("baduser", "badpass")
	basicAuthRouter.HandleContext(testContext)
	suite.Equal(401, testContext.Writer.Status())

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/", nil)
	testContext.Request.SetBasicAuth("testuser", "testpass")
	basicAuthRouter.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())

	// Test basic auth (anonymous get)
	basicAuthRouterAnonGet := NewRouter(RouterOptions{
		Logger:       log,
		Depth:        0,
		Username:     "testuser",
		Password:     "testpass",
		AnonymousGet: true,
	})
	basicAuthRouterAnonGet.SetRoutes(testRoutes)

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/", nil)
	basicAuthRouterAnonGet.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())
}

func (suite *RouterTestSuite) TestMapURLWithParamsBackToRouteTemplate() {
	tests := []struct {
		ctx    *gin.Context
		expect string
	}{
		{&gin.Context{
			Request: &http.Request{URL: &url.URL{Path: "/index.yaml"}},
		}, "/index.yaml"},
		{&gin.Context{
			Request: &http.Request{URL: &url.URL{Path: "/charts/foo-1.2.3.tgz"}},
			Params:  gin.Params{gin.Param{"filename", "foo-1.2.3.tgz"}},
		}, "/charts/:filename"},
		{&gin.Context{
			Request: &http.Request{URL: &url.URL{Path: "/api/charts/foo/1.2.3"}},
			Params:  gin.Params{gin.Param{"name", "foo"}, gin.Param{"version", "1.2.3"}},
		}, "/api/charts/:name/:version"},
		{&gin.Context{
			Request: &http.Request{URL: &url.URL{Path: "/api/charts/charts-repo/1.2.3+api"}},
			Params:  gin.Params{gin.Param{"name", "charts-repo"}, gin.Param{"version", "1.2.3+api"}},
		}, "/api/charts/:name/:version"},
		{&gin.Context{
			Request: &http.Request{URL: &url.URL{Path: "/api/charts/chart/1.2.3"}},
			Params:  gin.Params{gin.Param{"name", "chart"}, gin.Param{"version", "1.2.3"}},
		}, "/api/charts/:name/:version"},
		{&gin.Context{
			Request: &http.Request{URL: &url.URL{Path: "/api/charts/chart"}},
			Params:  gin.Params{gin.Param{"name", "chart"}},
		}, "/api/charts/:name"},
	}
	for _, tt := range tests {
		actual := mapURLWithParamsBackToRouteTemplate(tt.ctx)
		suite.Equal(tt.expect, actual)
	}
}

func TestRouterTestSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}
