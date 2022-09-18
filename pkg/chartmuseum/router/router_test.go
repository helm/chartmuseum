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
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/suite"

	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"

	"net/http/httptest"

	cm_auth "github.com/chartmuseum/auth"
	"github.com/gin-gonic/gin"
)

var (
	testPublicKey      = "../../../testdata/bearerauth/server.pem"
	testPrivateKey     = "../../../testdata/bearerauth/server.key"
	testClientAuthCert = "../../../testdata/clientauthcerts/server.pem"
	testClientAuthKey  = "../../../testdata/clientauthcerts/server.key"
	testClientAuthCA   = "../../../testdata/clientauthcerts/ca.pem"
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
		}, cm_auth.PullAction},
		{"GET", "/health", func(c *gin.Context) {
			c.Data(200, "text/html", []byte("200"))
		}, ""},
		{"GET", "/:repo/whatsmyrepo", func(c *gin.Context) {
			c.Data(200, "text/html", []byte(c.GetString("repo")))
		}, cm_auth.PullAction},
		{"GET", "/api/:repo/whatsmyrepo", func(c *gin.Context) {
			c.Data(200, "text/html", []byte(c.GetString("repo")))
		}, cm_auth.PullAction},
		{"POST", "/api/:repo/writetorepo", func(c *gin.Context) {
			c.Data(200, "text/html", []byte(c.GetString("repo")))
		}, cm_auth.PushAction},
		{"GET", "/api/:repo/systemstats", func(c *gin.Context) {
			c.Data(200, "text/html", []byte(c.GetString("repo")))
		}, cm_auth.PullAction},
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

	// Client Certificate Auth
	clientAuthRouter := NewRouter(RouterOptions{
		Logger:    log,
		TlsKey:    testClientAuthKey,
		TlsCert:   testClientAuthCert,
		TlsCACert: testClientAuthCA,
	})
	clientAuthRouter.SetRoutes(testRoutes)

	// Able to pull org1/repo1
	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/", nil)
	clientAuthRouter.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())

	// Bearer auth
	bearerAuthRouter := NewRouter(RouterOptions{
		Logger:       log,
		Depth:        2,
		BearerAuth:   true,
		AuthRealm:    "https://my.site.io/oauth2/token",
		AuthService:  "my.site.io",
		AuthCertPath: testPublicKey,
	})
	bearerAuthRouter.SetRoutes(testRoutes)

	// Generate a JWT token that has pull access to org1/repo1
	access := []cm_auth.AccessEntry{
		{
			Name:    "org1/repo1",
			Type:    cm_auth.AccessEntryType,
			Actions: []string{cm_auth.PullAction},
		},
	}
	cmtokenGenerator, err := cm_auth.NewTokenGenerator(&cm_auth.TokenGeneratorOptions{
		PrivateKeyPath: testPrivateKey,
	})
	suite.Nil(err)

	signedString, err := cmtokenGenerator.GenerateToken(access, 0)
	suite.Nil(err)

	// Able to pull org1/repo1
	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/org1/repo1/whatsmyrepo", nil)
	testContext.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", signedString))
	bearerAuthRouter.HandleContext(testContext)
	suite.Equal(200, testContext.Writer.Status())

	// Unable to push org1/repo1
	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("POST", "/api/org1/repo1/writetorepo", nil)
	testContext.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", signedString))
	bearerAuthRouter.HandleContext(testContext)
	suite.Equal(401, testContext.Writer.Status())

	// Cannot pull other repo (org1/repo2)
	testContext, _ = gin.CreateTestContext(httptest.NewRecorder())
	testContext.Request, _ = http.NewRequest("GET", "/org1/repo2/whatsmyrepo", nil)
	testContext.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", signedString))
	bearerAuthRouter.HandleContext(testContext)
	suite.Equal(401, testContext.Writer.Status())
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
			Params:  gin.Params{gin.Param{Key: "filename", Value: "foo-1.2.3.tgz"}},
		}, "/charts/:filename"},
		{&gin.Context{
			Request: &http.Request{URL: &url.URL{Path: "/api/charts/foo/1.2.3"}},
			Params:  gin.Params{gin.Param{Key: "name", Value: "foo"}, gin.Param{Key: "version", Value: "1.2.3"}},
		}, "/api/charts/:name/:version"},
		{&gin.Context{
			Request: &http.Request{URL: &url.URL{Path: "/api/charts/charts-repo/1.2.3+api"}},
			Params:  gin.Params{gin.Param{Key: "name", Value: "charts-repo"}, gin.Param{Key: "version", Value: "1.2.3+api"}},
		}, "/api/charts/:name/:version"},
		{&gin.Context{
			Request: &http.Request{URL: &url.URL{Path: "/api/charts/chart/1.2.3"}},
			Params:  gin.Params{gin.Param{Key: "name", Value: "chart"}, gin.Param{Key: "version", Value: "1.2.3"}},
		}, "/api/charts/:name/:version"},
		{&gin.Context{
			Request: &http.Request{URL: &url.URL{Path: "/api/charts/chart"}},
			Params:  gin.Params{gin.Param{Key: "name", Value: "chart"}},
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
