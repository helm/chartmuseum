package multitenant

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chartmuseum/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"

	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_router "helm.sh/chartmuseum/pkg/chartmuseum/router"
	cm_repo "helm.sh/chartmuseum/pkg/repo"

	"helm.sh/helm/v3/pkg/chart"
	helm_repo "helm.sh/helm/v3/pkg/repo"
)

type HandlerTestSuite struct {
	suite.Suite
	ServerDepth0  *MultiTenantServer
	TempDirectory string
}

func (suite *HandlerTestSuite) getServer(depth int) *MultiTenantServer {
	logger, err := cm_logger.NewLogger(cm_logger.LoggerOptions{
		Debug: true,
	})
	suite.Nil(err, "no error creating logger")

	backend := storage.Backend(storage.NewLocalFilesystemBackend(os.TempDir()))
	router := cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         depth,
		EnableMetrics: true,
		MaxUploadSize: maxUploadSize,
	})
	serverDepth, err := NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		ChartPostFormFieldName: "chart",
		ProvPostFormFieldName:  "prov",
		IndexLimit:             1,
		CacheInterval:          time.Second,
	})
	suite.NoError(err, "no error creating server")
	return serverDepth
}

func (suite *HandlerTestSuite) TestCustomStaticFilesHandler() {
	recorder := httptest.NewRecorder()
	testContext, _ := gin.CreateTestContext(recorder)
	testContext.Request, _ = http.NewRequest("GET", "/static/main.css", nil)
	suite.ServerDepth0 = suite.getServer(0)
	suite.ServerDepth0.WebTemplatePath = "testdata/template"
	suite.ServerDepth0.getStaticFilesHandler(testContext)
	data, err := os.ReadFile("testdata/template/static/main.css")
	if err != nil {
		suite.Fail("could not read testdata/template/static/main.css")
	}
	suite.Equal(200, recorder.Result().StatusCode)
	suite.Equal("text/css; charset=utf-8", recorder.Header().Get("Content-Type"))
	suite.Equal(string(data), recorder.Body.String())
}

func (suite *HandlerTestSuite) TestDefaultStaticFilesHandler() {
	recorder := httptest.NewRecorder()
	testContext, _ := gin.CreateTestContext(recorder)
	testContext.Request, _ = http.NewRequest("GET", "/static/main.css", nil)
	suite.ServerDepth0 = suite.getServer(0)
	suite.ServerDepth0.getStaticFilesHandler(testContext)
	suite.Equal(200, recorder.Result().StatusCode)
	suite.Equal("", recorder.Header().Get("Content-Type"))
}

func (suite *HandlerTestSuite) TestCustomWelcomePage() {
	recorder := httptest.NewRecorder()
	testContext, engine := gin.CreateTestContext(recorder)

	testContext.Request, _ = http.NewRequest("GET", "/", nil)
	suite.ServerDepth0 = suite.getServer(0)
	suite.ServerDepth0.WebTemplatePath = "testdata/template"
	suite.ServerDepth0.Router.Engine = engine
	suite.ServerDepth0.Router.LoadHTMLGlob(fmt.Sprintf("%s/*.html", suite.ServerDepth0.WebTemplatePath))
	suite.ServerDepth0.getWelcomePageHandler(testContext)
	data, err := os.ReadFile("testdata/template/index.html")
	if err != nil {
		suite.Fail("could not read testdata/template/index.html")
	}
	suite.Equal(200, recorder.Result().StatusCode)
	suite.Equal("text/html; charset=utf-8", recorder.Header().Get("Content-Type"))
	suite.Equal(string(data), recorder.Body.String())
}

func (suite *HandlerTestSuite) TestDefaultWelcomePage() {
	recorder := httptest.NewRecorder()
	testContext, _ := gin.CreateTestContext(recorder)
	testContext.Request, _ = http.NewRequest("GET", "/", nil)
	suite.ServerDepth0 = suite.getServer(0)
	suite.ServerDepth0.getWelcomePageHandler(testContext)
	data, err := os.ReadFile("testdata/default/index.html")
	if err != nil {
		suite.Fail("could not read testdata/default/index.html")
	}
	suite.Equal(200, recorder.Result().StatusCode)
	suite.Equal("text/html", recorder.Header().Get("Content-Type"))
	suite.Equal(string(data), recorder.Body.String())
}

func (suite *HandlerTestSuite) TestMissingTemplatesWelcomePage() {
	recorder := httptest.NewRecorder()
	testContext, engine := gin.CreateTestContext(recorder)
	testContext.Request, _ = http.NewRequest("GET", "/", nil)
	suite.ServerDepth0 = suite.getServer(0)
	suite.ServerDepth0.WebTemplatePath = "testdata/dummy"
	suite.ServerDepth0.Router.Engine = engine
	suite.ServerDepth0.getWelcomePageHandler(testContext)
	data, err := os.ReadFile("testdata/default/index.html")
	if err != nil {
		suite.Fail("could not read testdata/default/index.html")
	}
	suite.Equal(200, recorder.Result().StatusCode)
	suite.Equal("text/html", recorder.Header().Get("Content-Type"))
	suite.Equal(string(data), recorder.Body.String())
}

func (suite *HandlerTestSuite) TestGetAllChartsPagination() {
	server := suite.getServer(0)
	index := cm_repo.NewIndex("", "", &cm_repo.ServerInfo{}, false)
	for _, name := range []string{"a", "b", "c", "d", "e"} {
		index.AddEntry(&helm_repo.ChartVersion{
			Metadata: &chart.Metadata{Name: name, Version: "0.1.0"},
			URLs:     []string{fmt.Sprintf("charts/%s-0.1.0.tgz", name)},
		})
	}
	server.InternalCacheStore.Store("", &cacheEntry{RepoName: "", RepoIndex: index})

	tests := []struct {
		query string
		want  []string
	}{
		{"", []string{"a", "b", "c", "d", "e"}},
		{"?limit=2", []string{"a", "b"}},
		{"?offset=1&limit=2", []string{"b", "c"}},
		{"?offset=2", []string{"c", "d", "e"}},
		{"?offset=4", []string{"e"}},
		{"?offset=5", []string{}},
		{"?offset=9&limit=2", []string{}},
		{"?offset=3&limit=10", []string{"d", "e"}},
		{fmt.Sprintf("?offset=1&limit=%d", math.MaxInt), []string{"b", "c", "d", "e"}},
	}

	for _, tt := range tests {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request, _ = http.NewRequest("GET", "/api/charts"+tt.query, nil)
		server.getAllChartsRequestHandler(c)
		suite.Equal(200, recorder.Result().StatusCode, tt.query)

		var got map[string]helm_repo.ChartVersions
		suite.NoError(json.Unmarshal(recorder.Body.Bytes(), &got), tt.query)
		names := []string{}
		for name := range got {
			names = append(names, name)
		}
		suite.ElementsMatch(tt.want, names, "GET /api/charts%s", tt.query)
	}
}

func TestHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}
