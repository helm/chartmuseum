package multitenant

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	pathutil "path"
	"testing"
	"time"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

var testTarballPath = "../../../../testdata/charts/mychart/mychart-0.1.0.tgz"
var testProvfilePath = "../../../../testdata/charts/mychart/mychart-0.1.0.tgz.prov"

type MultiTenantServerTestSuite struct {
	suite.Suite
	Depth0Server         *MultiTenantServer
	Depth1Server         *MultiTenantServer
	Depth2Server         *MultiTenantServer
	Depth3Server         *MultiTenantServer
	TempDirectory        string
	TestTarballFilename  string
	TestProvfileFilename string
	StorageDirectory     map[string]map[string][]string
	LastCrashMessage     string
	LastPrinted          string
	LastExitCode         int
}

func (suite *MultiTenantServerTestSuite) doRequest(stype string, method string, urlStr string, body io.Reader, contentType string) gin.ResponseWriter {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest(method, urlStr, body)
	if contentType != "" {
		c.Request.Header.Set("Content-Type", contentType)
	}

	switch stype {
	case "depth0":
		suite.Depth0Server.Router.HandleContext(c)
	case "depth1":
		suite.Depth1Server.Router.HandleContext(c)
	case "depth2":
		suite.Depth2Server.Router.HandleContext(c)
	case "depth3":
		suite.Depth3Server.Router.HandleContext(c)
	}

	return c.Writer
}

func (suite *MultiTenantServerTestSuite) copyTestFilesTo(dir string) {
	srcFileTarball, err := os.Open(testTarballPath)
	suite.Nil(err, "no error opening test tarball")
	defer srcFileTarball.Close()

	destFileTarball, err := os.Create(pathutil.Join(dir, "mychart-0.1.0.tgz"))
	suite.Nil(err, fmt.Sprintf("no error creating new tarball in %s", dir))
	defer destFileTarball.Close()

	_, err = io.Copy(destFileTarball, srcFileTarball)
	suite.Nil(err, fmt.Sprintf("no error copying test testball to %s", dir))

	err = destFileTarball.Sync()
	suite.Nil(err, fmt.Sprintf("no error syncing temp tarball in %s", dir))

	srcFileProvfile, err := os.Open(testProvfilePath)
	suite.Nil(err, "no error opening test provenance file")
	defer srcFileProvfile.Close()

	destFileProvfile, err := os.Create(pathutil.Join(dir, "mychart-0.1.0.tgz.prov"))
	suite.Nil(err, fmt.Sprintf("no error creating new provenance file in %s", dir))
	defer destFileProvfile.Close()

	_, err = io.Copy(destFileProvfile, srcFileProvfile)
	suite.Nil(err, fmt.Sprintf("no error copying test provenance file to %s", dir))

	err = destFileProvfile.Sync()
	suite.Nil(err, fmt.Sprintf("no error syncing temp provenance file in %s", dir))
}

func (suite *MultiTenantServerTestSuite) populateOrgTeamRepoDirectory(org string, team string, repo string) {
	testPrefix := fmt.Sprintf("%s/%s/%s", org, team, repo)
	newDir := pathutil.Join(suite.TempDirectory, testPrefix)
	os.MkdirAll(newDir, os.ModePerm)
	suite.copyTestFilesTo(newDir)
	suite.copyTestFilesTo(pathutil.Join(newDir, ".."))
	suite.copyTestFilesTo(pathutil.Join(newDir, "../.."))
}

func (suite *MultiTenantServerTestSuite) SetupSuite() {
	echo = func(v ...interface{}) (int, error) {
		suite.LastPrinted = fmt.Sprint(v...)
		return 0, nil
	}

	exit = func(code int) {
		suite.LastExitCode = code
		suite.LastCrashMessage = fmt.Sprintf("exited %d", code)
	}

	timestamp := time.Now().Format("20060102150405")
	suite.TempDirectory = fmt.Sprintf("../../../../.test/chartmuseum-multitenant-server/%s", timestamp)
	os.MkdirAll(suite.TempDirectory, os.ModePerm)
	suite.copyTestFilesTo(suite.TempDirectory)

	suite.StorageDirectory = map[string]map[string][]string{
		"org1": {
			"team1": {"repo1", "repo2", "repo3"},
			"team2": {"repo1", "repo2", "repo3"},
			"team3": {"repo1", "repo2", "repo3"},
		},
		"org2": {
			"team1": {"repo1", "repo2", "repo3"},
			"team2": {"repo1", "repo2", "repo3"},
			"team3": {"repo1", "repo2", "repo3"},
		},
		"org3": {
			"team1": {"repo1", "repo2", "repo3"},
			"team2": {"repo1", "repo2", "repo3"},
			"team3": {"repo1", "repo2", "repo3"},
		},
	}

	// Scaffold out test storage directory structure
	for org, teams := range suite.StorageDirectory {
		for team, repos := range teams {
			for _, repo := range repos {
				suite.populateOrgTeamRepoDirectory(org, team, repo)
			}
		}
	}

	backend := storage.Backend(storage.NewLocalFilesystemBackend(suite.TempDirectory))

	logger, err := cm_logger.NewLogger(cm_logger.LoggerOptions{
		Debug: true,
	})
	suite.Nil(err, "no error creating logger")

	router := cm_router.NewRouter(cm_router.RouterOptions{
		Logger: logger,
		Depth:  0,
	})
	server, err := NewMultiTenantServer(MultiTenantServerOptions{
		Logger:         logger,
		Router:         router,
		StorageBackend: backend,
		EnableAPI:      true,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new multitenant (depth=0) server")
	suite.Depth0Server = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger: logger,
		Depth:  1,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:         logger,
		Router:         router,
		StorageBackend: backend,
		EnableAPI:      true,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new multitenant (depth=1) server")
	suite.Depth1Server = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger: logger,
		Depth:  2,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:         logger,
		Router:         router,
		StorageBackend: backend,
		EnableAPI:      true,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new multitenant (depth=2) server")
	suite.Depth2Server = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger: logger,
		Depth:  3,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:         logger,
		Router:         router,
		StorageBackend: backend,
		EnableAPI:      true,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new multitenant (depth=3) server")
	suite.Depth3Server = server
}

func (suite *MultiTenantServerTestSuite) TearDownSuite() {
	err := os.RemoveAll(suite.TempDirectory)
	suite.Nil(err, "no error deleting temp directory for local storage")
}

func (suite *MultiTenantServerTestSuite) TestGenIndex() {
	logger, err := cm_logger.NewLogger(cm_logger.LoggerOptions{
		Debug:   true,
		LogJSON: true,
	})
	suite.Nil(err, "no error creating logger")

	router := cm_router.NewRouter(cm_router.RouterOptions{
		Logger: logger,
	})

	NewMultiTenantServer(MultiTenantServerOptions{
		Logger:         logger,
		Router:         router,
		StorageBackend: suite.Depth0Server.StorageBackend,
		GenIndex:       true,
	})
	suite.Equal("exited 0", suite.LastCrashMessage, "no error with --gen-index")
	suite.Equal(0, suite.LastExitCode, "--gen-index flag exits 0")
	suite.Contains(suite.LastPrinted, "apiVersion:", "--gen-index prints yaml")
}

func (suite *MultiTenantServerTestSuite) TestRoutes() {
	suite.testAllRoutes("", 0)
	for org, teams := range suite.StorageDirectory {
		suite.testAllRoutes(org, 1)
		for team, repos := range teams {
			suite.testAllRoutes(pathutil.Join(org, team), 2)
			for _, repo := range repos {
				suite.testAllRoutes(pathutil.Join(org, team, repo), 3)
			}
		}
	}
}

func (suite *MultiTenantServerTestSuite) testAllRoutes(repo string, depth int) {
	var res gin.ResponseWriter

	stype := fmt.Sprintf("depth%d", depth)

	// GET /
	res = suite.doRequest(stype, "GET", "/", nil, "")
	suite.Equal(200, res.Status(), "200 GET /")

	// GET /health
	res = suite.doRequest(stype, "GET", "/health", nil, "")
	suite.Equal(200, res.Status(), "200 GET /health")

	var repoPrefix string
	if repo != "" {
		repoPrefix = pathutil.Join("/", repo)
	} else {
		repoPrefix = ""
	}

	// GET /:repo/index.yaml
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/index.yaml", repoPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/index.yaml", repoPrefix))

	// GET /:repo/charts/:filename
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart-0.1.0.tgz", repoPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart-0.1.0.tgz", repoPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart-0.1.0.tgz.prov", repoPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart-0.1.0.tgz.prov", repoPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/fakechart-0.1.0.tgz", repoPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("404 GET %s/charts/fakechart-0.1.0.tgz", repoPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/fakechart-0.1.0.tgz.prov", repo), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("404 GET %s/charts/fakechart-0.1.0.tgz.prov", repo))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/fakechart-0.1.0.bad", repo), nil, "")
	suite.Equal(500, res.Status(), fmt.Sprintf("500 GET %s/charts/fakechart-0.1.0.bad", repo))

	apiPrefix := pathutil.Join("/api", repo)

	// GET /api/charts
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts", apiPrefix))

	// GET /api/charts/:name
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/fakechart", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("404 GET %s/charts/fakechart", apiPrefix))

	// GET /api/charts/:name/:version
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart/0.1.0", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart/0.1.0", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart/latest", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart/latest", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart/0.1.1", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart/0.1.1", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/fakechart/0.1.0", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("200 GET %s/charts/fakechart/0.1.0", apiPrefix))
}

func TestMultiTenantServerTestSuite(t *testing.T) {
	suite.Run(t, new(MultiTenantServerTestSuite))
}
