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

	"github.com/kubernetes-helm/chartmuseum/pkg/cache"
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
	Server               *MultiTenantServer
	TempDirectory        string
	TestTarballFilename  string
	TestProvfileFilename string
	StorageDirectory     map[string][]string
}

func (suite *MultiTenantServerTestSuite) doRequest(stype string, method string, urlStr string, body io.Reader, contentType string) gin.ResponseWriter {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest(method, urlStr, body)
	if contentType != "" {
		c.Request.Header.Set("Content-Type", contentType)
	}

	switch stype {
	case "anonymous":
		suite.Server.Router.HandleContext(c)
	}

	return c.Writer
}

func (suite *MultiTenantServerTestSuite) populateOrgRepoDirectory(org string, repo string) {
	testPrefix := fmt.Sprintf("%s/%s", org, repo)
	newDir := pathutil.Join(suite.TempDirectory, testPrefix)
	os.MkdirAll(newDir, os.ModePerm)

	srcFileTarball, err := os.Open(testTarballPath)
	suite.Nil(err, "no error opening test tarball")
	defer srcFileTarball.Close()

	destFileTarball, err := os.Create(pathutil.Join(newDir, "mychart-0.1.0.tgz"))
	suite.Nil(err, fmt.Sprintf("no error creating new tarball in %s", testPrefix))
	defer destFileTarball.Close()

	_, err = io.Copy(destFileTarball, srcFileTarball)
	suite.Nil(err, fmt.Sprintf("no error copying test testball to %s", testPrefix))

	err = destFileTarball.Sync()
	suite.Nil(err, fmt.Sprintf("no error syncing temp tarball in %s", testPrefix))

	srcFileProvfile, err := os.Open(testProvfilePath)
	suite.Nil(err, "no error opening test provenance file")
	defer srcFileProvfile.Close()

	destFileProvfile, err := os.Create(pathutil.Join(newDir, "mychart-0.1.0.tgz.prov"))
	suite.Nil(err, fmt.Sprintf("no error creating new provenance file in %s", testPrefix))
	defer destFileTarball.Close()

	_, err = io.Copy(destFileProvfile, srcFileProvfile)
	suite.Nil(err, fmt.Sprintf("no error copying test provenance file to %s", testPrefix))

	err = destFileProvfile.Sync()
	suite.Nil(err, fmt.Sprintf("no error syncing temp provenance file in %s", testPrefix))
}

func (suite *MultiTenantServerTestSuite) SetupSuite() {
	timestamp := time.Now().Format("20060102150405")
	suite.TempDirectory = fmt.Sprintf("../../../../.test/chartmuseum-multitenant-server/%s", timestamp)

	suite.StorageDirectory = map[string][]string{
		"org1": {"repo1", "repo2", "repo3"},
		"org2": {"repo1", "repo2", "repo3"},
		"org3": {"repo1", "repo2", "repo3"},
	}

	// Scaffold out test storage directory structure
	for org, repos := range suite.StorageDirectory {
		for _, repo := range repos {
			suite.populateOrgRepoDirectory(org, repo)
		}
	}

	backend := storage.Backend(storage.NewLocalFilesystemBackend(suite.TempDirectory))

	logger, err := cm_logger.NewLogger(cm_logger.LoggerOptions{})
	suite.Nil(err, "no error creating logger")

	router := cm_router.NewRouter(cm_router.RouterOptions{
		Logger:     logger,
		PathPrefix: PathPrefix,
	})

	server, err := NewMultiTenantServer(MultiTenantServerOptions{
		Logger:         logger,
		Router:         router,
		StorageBackend: backend,
		Cache:          cache.NewInMemoryStore(),
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new multitenant server")

	suite.Server = server
}

func (suite *MultiTenantServerTestSuite) TearDownSuite() {
	err := os.RemoveAll(suite.TempDirectory)
	suite.Nil(err, "no error deleting temp directory for local storage")
}

func (suite *MultiTenantServerTestSuite) TestRoutes() {
	var res gin.ResponseWriter

	// GET /
	res = suite.doRequest("anonymous", "GET", "/", nil, "")
	suite.Equal(200, res.Status(), "200 GET /")

	for org, repos := range suite.StorageDirectory {
		for _, repo := range repos {
			prefix := fmt.Sprintf("%s/%s", org, repo)

			// GET /<org>/<repo>/index.yaml
			res = suite.doRequest("anonymous", "GET", fmt.Sprintf("/%s/index.yaml", prefix), nil, "")
			suite.Equal(200, res.Status(), fmt.Sprintf("200 GET /%s/index.yaml", prefix))

			// GET /<org>/<repo>/charts/<filename>
			res = suite.doRequest("anonymous", "GET", fmt.Sprintf("/%s/charts/mychart-0.1.0.tgz", prefix), nil, "")
			suite.Equal(200, res.Status(), fmt.Sprintf("200 GET /%s/charts/mychart-0.1.0.tgz", prefix))

			res = suite.doRequest("anonymous", "GET", fmt.Sprintf("/%s/charts/mychart-0.1.0.tgz.prov", prefix), nil, "")
			suite.Equal(200, res.Status(), fmt.Sprintf("200 GET /%s/charts/mychart-0.1.0.tgz.prov", prefix))

			res = suite.doRequest("anonymous", "GET", fmt.Sprintf("/%s/charts/fakechart-0.1.0.tgz", prefix), nil, "")
			suite.Equal(404, res.Status(), fmt.Sprintf("404 GET /%s/charts/fakechart-0.1.0.tgz", prefix))

			res = suite.doRequest("anonymous", "GET", fmt.Sprintf("/%s/charts/fakechart-0.1.0.tgz.prov", prefix), nil, "")
			suite.Equal(404, res.Status(), fmt.Sprintf("404 GET /%s/charts/fakechart-0.1.0.tgz.prov", prefix))

			res = suite.doRequest("anonymous", "GET", fmt.Sprintf("/%s/charts/fakechart-0.1.0.bad", prefix), nil, "")
			suite.Equal(500, res.Status(), fmt.Sprintf("500 GET /%s/charts/fakechart-0.1.0.bad", prefix))
		}
	}
}

func TestMultiTenantServerTestSuite(t *testing.T) {
	suite.Run(t, new(MultiTenantServerTestSuite))
}
