package router

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"

	"github.com/gin-gonic/gin"
	"net/http/httptest"
)

type RouterTestSuite struct {
	suite.Suite
}

func (suite *RouterTestSuite) TestRouterHandleContext() {
	logger, err := logger.NewLogger(true, true)
	suite.Nil(err, "no error creating logger")

	routerMetricsDisabled := NewRouter(logger, false)
	testContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/", nil)
	routerMetricsDisabled.HandleContext(testContext)
	suite.Equal(404, testContext.Writer.Status())

	routerMetricsEnabled := NewRouter(logger, true)
	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/", nil)
	routerMetricsEnabled.HandleContext(testContext)
	suite.Equal(404, testContext.Writer.Status())

	routerMetricsEnabled.GET("/giveme200", func (c *gin.Context) {
		c.Data(200, "text/html", []byte("200"))
	})

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/giveme200", nil)
	routerMetricsEnabled.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())

	routerMetricsEnabled.GET("/giveme500", func (c *gin.Context) {
		c.Data(500, "text/html", []byte("500"))
	})

	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/giveme500", nil)
	routerMetricsEnabled.HandleContext(testContext)
	suite.Equal(500, testContext.Writer.Status())
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
