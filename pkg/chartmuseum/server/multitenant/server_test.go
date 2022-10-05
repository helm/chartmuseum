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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	pathutil "path"
	"strings"
	"testing"
	"time"

	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_router "helm.sh/chartmuseum/pkg/chartmuseum/router"
	"helm.sh/chartmuseum/pkg/repo"

	"github.com/chartmuseum/storage"
	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

var maxUploadSize = 1024 * 1024 * 20

// These are generated from scripts/setup-test-environment.sh
var testTarballPath = "../../../../testdata/charts/mychart/mychart-0.1.0.tgz"
var testTarballPathV2 = "../../../../testdata/charts/mychart/mychart-0.2.0.tgz"
var testTarballPathV0 = "../../../../testdata/charts/mychart/mychart-0.0.1.tgz"
var testProvfilePath = "../../../../testdata/charts/mychart/mychart-0.1.0.tgz.prov"
var otherTestTarballPath = "../../../../testdata/charts/otherchart/otherchart-0.1.0.tgz"
var otherTestProvfilePath = "../../../../testdata/charts/otherchart/otherchart-0.1.0.tgz.prov"
var badTestTarballPath = "../../../../testdata/badcharts/mybadchart/mybadchart-1.0.0.tgz"
var badTestProvfilePath = "../../../../testdata/badcharts/mybadchart/mybadchart-1.0.0.tgz.prov"

type MultiTenantServerTestSuite struct {
	suite.Suite
	Depth0Server            *MultiTenantServer
	Depth1Server            *MultiTenantServer
	Depth2Server            *MultiTenantServer
	Depth3Server            *MultiTenantServer
	DisabledAPIServer       *MultiTenantServer
	DisabledDeleteServer    *MultiTenantServer
	OverwriteServer         *MultiTenantServer
	ForceOverwriteServer    *MultiTenantServer
	ChartURLServer          *MultiTenantServer
	MaxObjectsServer        *MultiTenantServer
	MaxUploadSizeServer     *MultiTenantServer
	Semver2Server           *MultiTenantServer
	PerChartLimitServer     *MultiTenantServer
	ArtifactHubRepoIDServer *MultiTenantServer
	UpdateToDateServer      *MultiTenantServer
	CacheInternalServer     *MultiTenantServer
	TempDirectory           string
	TestTarballFilename     string
	TestProvfileFilename    string
	StorageDirectory        map[string]map[string][]string
	LastCrashMessage        string
	LastPrinted             string
	LastExitCode            int
	ArtifactHubIds          map[string]string
	AlwaysRegenerateIndex   bool
}

func (suite *MultiTenantServerTestSuite) doRequest(stype string, method string, urlStr string, body io.Reader, contentType string, output ...*bytes.Buffer) gin.ResponseWriter {
	recorder := httptest.NewRecorder()
	if len(output) > 0 {
		recorder.Body = output[0]
	}
	c, _ := gin.CreateTestContext(recorder)
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
	case "disabled":
		suite.DisabledAPIServer.Router.HandleContext(c)
	case "disableddelete":
		suite.DisabledDeleteServer.Router.HandleContext(c)
	case "overwrite":
		suite.OverwriteServer.Router.HandleContext(c)
	case "forceoverwrite":
		suite.ForceOverwriteServer.Router.HandleContext(c)
	case "charturl":
		suite.ChartURLServer.Router.HandleContext(c)
	case "maxobjects":
		suite.MaxObjectsServer.Router.HandleContext(c)
	case "maxuploadsize":
		suite.MaxUploadSizeServer.Router.HandleContext(c)
	case "semver2":
		suite.Semver2Server.Router.HandleContext(c)
	case "per-chart-limit":
		suite.PerChartLimitServer.Router.HandleContext(c)
	case "artifacthub":
		suite.ArtifactHubRepoIDServer.Router.HandleContext(c)
	case "chart-up-to-date":
		suite.UpdateToDateServer.Router.HandleContext(c)
	case "cache-interval":
		suite.CacheInternalServer.Router.HandleContext(c)
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

	srcFileTarball, err := os.Open(testTarballPath)
	suite.Nil(err, "no error opening test tarball")
	defer srcFileTarball.Close()

	suite.TestTarballFilename = pathutil.Join(suite.TempDirectory, "mychart-0.1.0.tgz")
	destFileTarball, err := os.Create(suite.TestTarballFilename)
	suite.Nil(err, "no error creating new tarball in temp dir")
	defer destFileTarball.Close()

	_, err = io.Copy(destFileTarball, srcFileTarball)
	suite.Nil(err, "no error copying test testball to temp tarball")

	err = destFileTarball.Sync()
	suite.Nil(err, "no error syncing temp tarball")

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

	// build the map of Artifact Hub Repo Ids from the StorageDirectory
	// so we can test the /:repo/artifact-hub.yml route in TestRoutes
	suite.ArtifactHubIds = map[string]string{"": "depth0"}
	for depth1, v := range suite.StorageDirectory {
		suite.ArtifactHubIds[depth1] = "depth1"
		for depth2, v := range v {
			suite.ArtifactHubIds[fmt.Sprintf("%s/%s", depth1, depth2)] = "depth2"
			for _, depth3 := range v {
				suite.ArtifactHubIds[fmt.Sprintf("%s/%s/%s", depth1, depth2, depth3)] = "depth3"
			}
		}
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
		Logger:        logger,
		Depth:         0,
		EnableMetrics: true,
		MaxUploadSize: maxUploadSize,
	})
	server, err := NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		ChartPostFormFieldName: "chart",
		ProvPostFormFieldName:  "prov",
		IndexLimit:             1,
		ArtifactHubRepoID:      suite.ArtifactHubIds,
		CacheInterval:          time.Second,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new multitenant (depth=0) server")
	suite.Depth0Server = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         1,
		EnableMetrics: true,
		MaxUploadSize: maxUploadSize,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		ChartPostFormFieldName: "chart",
		ProvPostFormFieldName:  "prov",
		ArtifactHubRepoID:      suite.ArtifactHubIds,
		CacheInterval:          time.Second,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new multitenant (depth=1) server")
	suite.Depth1Server = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         2,
		MaxUploadSize: maxUploadSize,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		ChartPostFormFieldName: "chart",
		ProvPostFormFieldName:  "prov",
		ArtifactHubRepoID:      suite.ArtifactHubIds,
		CacheInterval:          time.Second,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new multitenant (depth=2) server")
	suite.Depth2Server = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         3,
		MaxUploadSize: maxUploadSize,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		ChartPostFormFieldName: "chart",
		ProvPostFormFieldName:  "prov",
		ArtifactHubRepoID:      suite.ArtifactHubIds,
		CacheInterval:          time.Second,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new multitenant (depth=3) server")
	suite.Depth3Server = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         0,
		MaxUploadSize: maxUploadSize,
	})

	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:         logger,
		Router:         router,
		StorageBackend: backend,
		EnableAPI:      false,
		CacheInterval:  time.Second,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new disabled server")
	suite.DisabledAPIServer = server

	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger: logger,
		Router: cm_router.NewRouter(cm_router.RouterOptions{
			Logger:        logger,
			Depth:         0,
			MaxUploadSize: maxUploadSize,
		}),
		StorageBackend: backend,
		EnableAPI:      true,
		DisableDelete:  true,
		CacheInterval:  time.Second,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new disabled delete server")
	suite.DisabledDeleteServer = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         0,
		MaxUploadSize: maxUploadSize,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		AllowOverwrite:         true,
		ChartPostFormFieldName: "chart",
		ProvPostFormFieldName:  "prov",
		CacheInterval:          time.Second,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new overwrite server")
	suite.OverwriteServer = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         0,
		MaxUploadSize: maxUploadSize,
	})

	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		AllowOverwrite:         true,
		ChartPostFormFieldName: "chart",
		CacheInterval:          time.Second,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating semantic version server")
	suite.Semver2Server = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         0,
		MaxUploadSize: maxUploadSize,
	})

	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		AllowOverwrite:         true,
		ChartPostFormFieldName: "chart",
		CacheInterval:          time.Second,
		PerChartLimit:          2,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating per-chart-limit server")
	suite.PerChartLimitServer = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         0,
		MaxUploadSize: maxUploadSize,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		AllowForceOverwrite:    true,
		ChartPostFormFieldName: "chart",
		ProvPostFormFieldName:  "prov",
		CacheInterval:          time.Second,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new forceoverwrite server")
	suite.ForceOverwriteServer = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         0,
		MaxUploadSize: maxUploadSize,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		ChartPostFormFieldName: "chart",
		ProvPostFormFieldName:  "prov",
		ChartURL:               "https://chartmuseum.com",
		CacheInterval:          time.Second,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new custom chart URL server")
	suite.ChartURLServer = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         0,
		MaxUploadSize: maxUploadSize,
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		AllowOverwrite:         true,
		ChartPostFormFieldName: "chart",
		ProvPostFormFieldName:  "prov",
		MaxStorageObjects:      1,
		CacheInterval:          time.Second,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new max objects server")
	suite.MaxObjectsServer = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         0,
		MaxUploadSize: 1, // intentionally small
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		AllowOverwrite:         true,
		ChartPostFormFieldName: "chart",
		ProvPostFormFieldName:  "prov",
		CacheInterval:          time.Second,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new max upload size server")
	suite.MaxUploadSizeServer = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         0,
		MaxUploadSize: 1, // intentionally small
	})
	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		AllowOverwrite:         true,
		ChartPostFormFieldName: "chart",
		ProvPostFormFieldName:  "prov",
		ArtifactHubRepoID:      suite.ArtifactHubIds,
		CacheInterval:          time.Second,
	})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new artifact hub repo id server")
	suite.ArtifactHubRepoIDServer = server

	router = cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Depth:         0,
		MaxUploadSize: 1,
	})

	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		AllowOverwrite:         true,
		ChartPostFormFieldName: "chart",
		ProvPostFormFieldName:  "prov",
		CacheInterval:          time.Second,
		AlwaysRegenerateIndex:  true,
	})

	suite.NotNil(server)
	suite.Nil(err, "can not create server with keep chart always up to date")
	suite.UpdateToDateServer = server

	server, err = NewMultiTenantServer(MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         backend,
		TimestampTolerance:     time.Duration(0),
		EnableAPI:              true,
		AllowOverwrite:         true,
		ChartPostFormFieldName: "chart",
		ProvPostFormFieldName:  "prov",
		CacheInterval:          time.Second,
	})

	suite.NotNil(server)
	suite.Nil(err, "cannot create cache interval server")
	suite.CacheInternalServer = server
}

func (suite *MultiTenantServerTestSuite) TearDownSuite() {
	os.RemoveAll(suite.TempDirectory)
}

func (suite *MultiTenantServerTestSuite) regenerateRepositoryIndex(repo string, isFound bool) {
	server := suite.Depth0Server
	if repo != "" {
		server = suite.Depth1Server
	}
	log := server.Logger.ContextLoggingFn(&gin.Context{})

	entry, err := server.initCacheEntry(log, repo)
	suite.Nil(err, "no error on init cache entry")

	objects, err := server.fetchChartsInStorage(log, repo)
	if !isFound {
		suite.Equal(len(objects), 0)
		return
	}
	suite.Nil(err, "no error on fetchChartsInStorage")
	diff := storage.GetObjectSliceDiff(server.getRepoObjectSliceWithLock(entry), objects, server.TimestampTolerance)
	_, err = server.regenerateRepositoryIndexWorker(log, entry, diff)
	suite.Nil(err, "no error regenerating repo index")

	newtime := time.Now().Add(1 * time.Hour)
	err = os.Chtimes(suite.TestTarballFilename, newtime, newtime)
	suite.Nil(err, "no error changing modtime on temp file")

	objects, err = server.fetchChartsInStorage(log, repo)
	suite.Nil(err, "no error on fetchChartsInStorage")
	diff = storage.GetObjectSliceDiff(server.getRepoObjectSliceWithLock(entry), objects, server.TimestampTolerance)
	_, err = server.regenerateRepositoryIndexWorker(log, entry, diff)
	suite.Nil(err, "no error regenerating repo index with tarball updated")

	brokenTarballFilename := pathutil.Join(suite.TempDirectory, "brokenchart.tgz")
	destFile, err := os.Create(brokenTarballFilename)
	suite.Nil(err, "no error creating new broken tarball in temp dir")
	defer destFile.Close()
	objects, err = server.fetchChartsInStorage(log, repo)
	suite.Nil(err, "no error on fetchChartsInStorage")
	diff = storage.GetObjectSliceDiff(server.getRepoObjectSliceWithLock(entry), objects, server.TimestampTolerance)
	_, err = server.regenerateRepositoryIndexWorker(log, entry, diff)
	suite.Nil(err, "error not returned with broken tarball added")

	err = os.Chtimes(brokenTarballFilename, newtime, newtime)
	suite.Nil(err, "no error changing modtime on broken tarball")
	objects, err = server.fetchChartsInStorage(log, repo)
	suite.Nil(err, "no error on fetchChartsInStorage")
	diff = storage.GetObjectSliceDiff(server.getRepoObjectSliceWithLock(entry), objects, server.TimestampTolerance)
	_, err = server.regenerateRepositoryIndexWorker(log, entry, diff)
	suite.Nil(err, "error not returned with broken tarball updated")

	err = os.Remove(brokenTarballFilename)
	suite.Nil(err, "no error removing broken tarball")
	objects, err = server.fetchChartsInStorage(log, repo)
	suite.Nil(err, "no error on fetchChartsInStorage")
	diff = storage.GetObjectSliceDiff(server.getRepoObjectSliceWithLock(entry), objects, server.TimestampTolerance)
	_, err = server.regenerateRepositoryIndexWorker(log, entry, diff)
	suite.Nil(err, "error not returned with broken tarball removed")
}

func (suite *MultiTenantServerTestSuite) TestRegenerateRepositoryIndex() {
	suite.regenerateRepositoryIndex("", true)
	suite.regenerateRepositoryIndex("org1", true)
	suite.regenerateRepositoryIndex("not-set-org", false)
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
		CacheInterval:  time.Second,
	})
	suite.Equal("exited 0", suite.LastCrashMessage, "no error with --gen-index")
	suite.Equal(0, suite.LastExitCode, "--gen-index flag exits 0")
	suite.Contains(suite.LastPrinted, "apiVersion:", "--gen-index prints yaml")
}

func (suite *MultiTenantServerTestSuite) TestStatefiles() {
	logger, err := cm_logger.NewLogger(cm_logger.LoggerOptions{
		Debug:   true,
		LogJSON: true,
	})
	suite.Nil(err, "no error creating logger")

	router := cm_router.NewRouter(cm_router.RouterOptions{
		Logger: logger,
	})

	// add an example index-cache.yaml
	indexCacheFilePath := pathutil.Join(suite.TempDirectory, repo.StatefileFilename)
	content := []byte(`apiVersion: v1
entries:
  acs-engine-autoscaler:
  - name: acs-engine-autoscaler
    urls:
    - charts/acs-engine-autoscaler-2.1.2.tgz
    version: 2.1.2
generated: "2018-05-23T15:14:46-05:00"`)
	err = ioutil.WriteFile(indexCacheFilePath, content, 0644)
	suite.Nil(err, "no error creating test index-cache.yaml")
	defer os.Remove(indexCacheFilePath)

	NewMultiTenantServer(MultiTenantServerOptions{
		Logger:         logger,
		Router:         router,
		StorageBackend: suite.Depth0Server.StorageBackend,
		UseStatefiles:  true,
		GenIndex:       true,
		CacheInterval:  time.Second,
	})
	suite.Equal("exited 0", suite.LastCrashMessage, "no error with --gen-index")
	suite.Equal(0, suite.LastExitCode, "--gen-index flag exits 0")
	suite.Contains(suite.LastPrinted, "apiVersion:", "--gen-index prints yaml")

	// remove index-cache.yaml
	err = os.Remove(indexCacheFilePath)
	suite.Nil(err, "no error deleting test index-cache.yaml")

	// invalid, unparsable index-cache.yaml
	indexCacheFilePath = pathutil.Join(suite.TempDirectory, repo.StatefileFilename)
	content = []byte(`is this valid yaml? maybe. but its definitely not a valid index.yaml!`)
	err = ioutil.WriteFile(indexCacheFilePath, content, 0644)
	suite.Nil(err, "no error creating test index-cache.yaml")

	NewMultiTenantServer(MultiTenantServerOptions{
		Logger:         logger,
		Router:         router,
		StorageBackend: suite.Depth0Server.StorageBackend,
		UseStatefiles:  true,
		GenIndex:       true,
		CacheInterval:  time.Second,
	})
	suite.Equal("exited 0", suite.LastCrashMessage, "no error with --gen-index")
	suite.Equal(0, suite.LastExitCode, "--gen-index flag exits 0")
	suite.Contains(suite.LastPrinted, "apiVersion:", "--gen-index prints yaml")

	// remove index-cache.yaml
	err = os.Remove(indexCacheFilePath)
	suite.Nil(err, "no error deleting test index-cache.yaml")

	// no index-cache.yaml
	NewMultiTenantServer(MultiTenantServerOptions{
		Logger:         logger,
		Router:         router,
		StorageBackend: suite.Depth0Server.StorageBackend,
		UseStatefiles:  true,
		GenIndex:       true,
		CacheInterval:  time.Second,
	})
	suite.Equal("exited 0", suite.LastCrashMessage, "no error with --gen-index")
	suite.Equal(0, suite.LastExitCode, "--gen-index flag exits 0")
	suite.Contains(suite.LastPrinted, "apiVersion:", "--gen-index prints yaml")
}

func (suite *MultiTenantServerTestSuite) TestDisabledServer() {
	// Test that all /api routes disabled if EnableAPI=false
	res := suite.doRequest("disabled", "GET", "/api/charts", nil, "")
	suite.Equal(404, res.Status(), "404 GET /api/charts")

	res = suite.doRequest("disabled", "GET", "/api/charts/mychart", nil, "")
	suite.Equal(404, res.Status(), "404 GET /api/charts")

	res = suite.doRequest("disabled", "GET", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(404, res.Status(), "404 GET /api/charts")

	body := bytes.NewBuffer([]byte{})
	res = suite.doRequest("disabled", "POST", "/api/charts", body, "")
	suite.Equal(404, res.Status(), "404 POST /api/charts")

	body = bytes.NewBuffer([]byte{})
	res = suite.doRequest("disabled", "POST", "/api/prov", body, "")
	suite.Equal(404, res.Status(), "404 POST /api/prov")

	res = suite.doRequest("disabled", "DELETE", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(404, res.Status(), "404 DELETE /api/charts/mychart/0.1.0")
}

func (suite *MultiTenantServerTestSuite) TestDisabledDeleteServer() {
	res := suite.doRequest("disableddelete", "DELETE", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(404, res.Status(), "404 DELETE /api/charts/mychart/0.1.0")
}

func (suite *MultiTenantServerTestSuite) extractRepoEntryFromInternalCache(repo string) *cacheEntry {
	local, ok := suite.OverwriteServer.InternalCacheStore.Load(repo)
	if ok {
		return local
	}
	return nil
}

func (suite *MultiTenantServerTestSuite) TestOverwriteServer() {
	// Check if files can be overwritten
	content, err := ioutil.ReadFile(testTarballPath)
	suite.Nil(err, "no error opening test tarball")
	body := bytes.NewBuffer(content)
	res := suite.doRequest("overwrite", "POST", "/api/charts", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/charts")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("overwrite", "POST", "/api/charts", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/charts")

	{
		// waiting for the emit event
		// the event is transferred via a channel , here do a simple wait for not changing the original structure
		// only for testing purpose
		time.Sleep(time.Second)
		// depth: 0
		e := suite.extractRepoEntryFromInternalCache("")
		e.RepoLock.RLock()
		suite.Equal(1, len(e.RepoIndex.Entries), "overwrite entries validation")
		e.RepoLock.RUnlock()
	}

	content, err = ioutil.ReadFile(testProvfilePath)
	suite.Nil(err, "no error opening test provenance file")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("overwrite", "POST", "/api/prov", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/prov")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("overwrite", "POST", "/api/prov", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/prov")

	buf, w := suite.getBodyWithMultipartFormFiles([]string{"chart", "prov"}, []string{testTarballPath, testProvfilePath})
	res = suite.doRequest("overwrite", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), "201 POST /api/charts")
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart", "prov"}, []string{testTarballPath, testProvfilePath})
	res = suite.doRequest("overwrite", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), "201 POST /api/charts")
	{
		// the same as chart only case above
		time.Sleep(time.Second)
		// depth: 0
		e := suite.extractRepoEntryFromInternalCache("")
		e.RepoLock.RLock()
		suite.Equal(1, len(e.RepoIndex.Entries), "overwrite entries validation")
		e.RepoLock.RUnlock()
	}
}

func (suite *MultiTenantServerTestSuite) TestBadChartUpload() {
	content, err := ioutil.ReadFile(badTestTarballPath)
	suite.Nil(err, "no error opening test tarball")

	body := bytes.NewBuffer(content)
	res := suite.doRequest("depth0", "POST", "/api/charts", body, "")
	suite.Equal(400, res.Status(), "400 POST /api/charts")

	content, err = ioutil.ReadFile(badTestProvfilePath)
	suite.Nil(err, "no error opening test provenance file")

	body = bytes.NewBuffer(content)
	res = suite.doRequest("depth0", "POST", "/api/prov", body, "")
	suite.Equal(400, res.Status(), "400 POST /api/prov")

	buf, w := suite.getBodyWithMultipartFormFiles([]string{"chart", "prov"}, []string{badTestTarballPath, badTestProvfilePath})
	res = suite.doRequest("depth0", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(400, res.Status(), "400 POST /api/charts")
}

func (suite *MultiTenantServerTestSuite) TestForceOverwriteServer() {
	// Clear test repo to allow uploading again
	res := suite.doRequest("forceoverwrite", "DELETE", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(200, res.Status(), "200 DELETE /api/charts/mychart/0.1.0")

	// Check if files can be overwritten when ?force is on URL
	content, err := ioutil.ReadFile(testTarballPath)
	suite.Nil(err, "no error opening test tarball")
	body := bytes.NewBuffer(content)
	res = suite.doRequest("forceoverwrite", "POST", "/api/charts", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/charts")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("forceoverwrite", "POST", "/api/charts", body, "")
	suite.Equal(409, res.Status(), "409 POST /api/charts")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("forceoverwrite", "POST", "/api/charts?force", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/charts?force")

	content, err = ioutil.ReadFile(testProvfilePath)
	suite.Nil(err, "no error opening test provenance file")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("forceoverwrite", "POST", "/api/prov", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/prov")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("forceoverwrite", "POST", "/api/prov", body, "")
	suite.Equal(409, res.Status(), "409 POST /api/prov")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("forceoverwrite", "POST", "/api/prov?force", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/prov?force")

	// Clear test repo to allow uploading again
	res = suite.doRequest("forceoverwrite", "DELETE", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(200, res.Status(), "200 DELETE /api/charts/mychart/0.1.0")

	buf, w := suite.getBodyWithMultipartFormFiles([]string{"chart", "prov"}, []string{testTarballPath, testProvfilePath})
	res = suite.doRequest("forceoverwrite", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), "201 POST /api/charts")
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart", "prov"}, []string{testTarballPath, testProvfilePath})
	res = suite.doRequest("forceoverwrite", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(409, res.Status(), "409 POST /api/charts")
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart", "prov"}, []string{testTarballPath, testProvfilePath})
	res = suite.doRequest("forceoverwrite", "POST", "/api/charts?force", buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), "201 POST /api/charts?force")
}

func (suite *MultiTenantServerTestSuite) TestCustomChartURLServer() {
	res := suite.doRequest("charturl", "GET", "/index.yaml", nil, "")
	suite.Equal(200, res.Status(), "200 GET /index.yaml")
}

func (suite *MultiTenantServerTestSuite) TestMaxObjectsServer() {
	// Overwrites should still be allowed if limit is reached
	content, err := ioutil.ReadFile(testTarballPath)
	suite.Nil(err, "no error opening test tarball")
	body := bytes.NewBuffer(content)
	res := suite.doRequest("maxobjects", "POST", "/api/charts", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/charts")

	content, err = ioutil.ReadFile(testProvfilePath)
	suite.Nil(err, "no error opening test provenance file")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("maxobjects", "POST", "/api/prov", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/prov")

	// trigger error, reached max
	content, err = ioutil.ReadFile(otherTestTarballPath)
	suite.Nil(err, "no error opening other test tarball")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("maxobjects", "POST", "/api/charts", body, "")
	suite.Equal(507, res.Status(), "507 POST /api/charts")

	content, err = ioutil.ReadFile(otherTestProvfilePath)
	suite.Nil(err, "no error opening other test provenance file")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("maxobjects", "POST", "/api/prov", body, "")
	suite.Equal(507, res.Status(), "507 POST /api/prov")
}

func (suite *MultiTenantServerTestSuite) TestPerChartLimit() {
	ns := "per-chart-limit"
	content, err := ioutil.ReadFile(testTarballPathV0)
	suite.Nil(err, "no error opening test tarball")
	body := bytes.NewBuffer(content)
	res := suite.doRequest(ns, "POST", "/api/charts", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/charts")

	content, err = ioutil.ReadFile(testTarballPathV2)
	suite.Nil(err, "no error opening test tarball")
	body = bytes.NewBuffer(content)
	res = suite.doRequest(ns, "POST", "/api/charts", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/charts")

	content, err = ioutil.ReadFile(testTarballPath)
	suite.Nil(err, "no error opening test tarball")
	body = bytes.NewBuffer(content)
	res = suite.doRequest(ns, "POST", "/api/charts", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/charts")

	time.Sleep(time.Second)

	res = suite.doRequest(ns, "GET", "/api/charts/mychart/0.2.0", nil, "")
	suite.Equal(200, res.Status(), "200 GET /api/charts/mychart-0.2.0")

	res = suite.doRequest(ns, "GET", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(200, res.Status(), "200 GET /api/charts/mychart-0.1.0")

	res = suite.doRequest(ns, "GET", "/api/charts/mychart/0.0.1", nil, "")
	suite.Equal(404, res.Status(), "200 GET /api/charts/mychart-0.0.1")
}

func (suite *MultiTenantServerTestSuite) TestMaxUploadSizeServer() {
	// trigger 413s, "request too large"
	content, err := ioutil.ReadFile(testTarballPath)
	suite.Nil(err, "no error opening test tarball")
	body := bytes.NewBuffer(content)
	res := suite.doRequest("maxuploadsize", "POST", "/api/charts", body, "")
	suite.Equal(413, res.Status(), "413 POST /api/charts")

	content, err = ioutil.ReadFile(testProvfilePath)
	suite.Nil(err, "no error opening test provenance file")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("maxuploadsize", "POST", "/api/prov", body, "")
	suite.Equal(413, res.Status(), "201 POST /api/prov")

	buf, w := suite.getBodyWithMultipartFormFiles([]string{"chart", "prov"}, []string{testTarballPath, testProvfilePath})
	res = suite.doRequest("maxuploadsize", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(413, res.Status(), "413 POST /api/charts")
}

func (suite *MultiTenantServerTestSuite) TestMetrics() {

	apiPrefix := pathutil.Join("/api", "a")

	content, err := ioutil.ReadFile(testTarballPath)
	suite.Nil(err, "error opening test tarball")

	body := bytes.NewBuffer(content)
	res := suite.doRequest("depth1", "POST", fmt.Sprintf("%s/charts", apiPrefix), body, "")
	suite.Equal(201, res.Status(), fmt.Sprintf("201 post %s/charts", apiPrefix))

	otherChart, err := ioutil.ReadFile(testTarballPathV2)
	suite.Nil(err, "error opening test tarball")

	body = bytes.NewBuffer(otherChart)
	res = suite.doRequest("depth1", "POST", fmt.Sprintf("%s/charts", apiPrefix), body, "")
	suite.Equal(201, res.Status(), fmt.Sprintf("201 POST %s/charts", apiPrefix))

	// GET /a/index.yaml to regenerate index (and metrics)
	res = suite.doRequest("depth1", "GET", "/a/index.yaml", nil, "")
	suite.Equal(200, res.Status(), "200 GET /a/index.yaml")

	// GET /b/index.yaml to regenerate b index (and metrics)
	res = suite.doRequest("depth1", "GET", "/b/index.yaml", nil, "")
	suite.Equal(200, res.Status(), "200 GET /b/index.yaml")

	// Get metrics
	buffer := bytes.NewBufferString("")
	res = suite.doRequest("depth1", "GET", "/metrics", nil, "", buffer)
	suite.Equal(200, res.Status(), "200 GET /metrics")

	metrics := buffer.String()
	//fmt.Print(metrics) // observe the metric output

	// Ensure that we have the Gauges as documented
	suite.True(strings.Contains(metrics, "# TYPE chartmuseum_chart_versions_served_total gauge"))
	suite.True(strings.Contains(metrics, "# TYPE chartmuseum_charts_served_total gauge"))

	suite.True(strings.Contains(metrics, "chartmuseum_charts_served_total{repo=\"a\"} 1"))
	suite.True(strings.Contains(metrics, "chartmuseum_chart_versions_served_total{repo=\"a\"} 2"))

	// Ensure that the b repo has no charts
	suite.True(strings.Contains(metrics, "chartmuseum_chart_versions_served_total{repo=\"b\"} 0"))
}

func (suite *MultiTenantServerTestSuite) TestAlwaysUpToDateChart() {
	res := suite.doRequest("chart-up-to-date", "GET", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(200, res.Status(), "200 GET /api/charts/mychart-0.1.0")
}

func (suite *MultiTenantServerTestSuite) TestCacheInterval() {
	res := suite.doRequest("cache-interval", "GET", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(200, res.Status(), "200 GET /api/charts/mychart-0.1.0")
}

func (suite *MultiTenantServerTestSuite) TestArtifactHubRepoID() {
	buffer := bytes.NewBufferString("")
	res := suite.doRequest("artifacthub", "GET", "/artifacthub-repo.yml", nil, "", buffer)
	suite.Equal(200, res.Status(), "200 GET /artifacthub-repo.yml")

	artifactHubYmlString := buffer.Bytes()
	artifactHubYmlFile := &repo.ArtifactHubFile{}
	yaml.Unmarshal(artifactHubYmlString, artifactHubYmlFile)
	suite.Equal(artifactHubYmlFile.RepoID, suite.ArtifactHubIds[""])
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

	// HEAD /:repo/index.yaml
	res = suite.doRequest(stype, "HEAD", fmt.Sprintf("%s/index.yaml", repoPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 HEAD %s/index.yaml", repoPrefix))

	// GET /:repo/artifacthub-repo.yaml
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/artifacthub-repo.yml", repoPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/artifacthub-repo.yml", repoPrefix))

	// Issue #21
	suite.NotEqual("", res.Header().Get("X-Request-Id"), "X-Request-Id header is present")
	suite.Equal("", res.Header().Get("X-Blah-Blah-Blah"), "X-Blah-Blah-Blah header is not present")

	// GET /:repo/charts/:filename
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart-0.1.0.tgz", repoPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart-0.1.0.tgz", repoPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart-0.1.0.tgz.prov", repoPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart-0.1.0.tgz.prov", repoPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/fakechart-0.1.0.tgz", repoPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("404 GET %s/charts/fakechart-0.1.0.tgz", repoPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/fakechart-0.1.0.tgz.prov", repoPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("404 GET %s/charts/fakechart-0.1.0.tgz.prov", repoPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/fakechart-0.1.0.bad", repoPrefix), nil, "")
	suite.Equal(500, res.Status(), fmt.Sprintf("500 GET %s/charts/fakechart-0.1.0.bad", repoPrefix))

	apiPrefix := pathutil.Join("/api", repo)

	// GET /api/:repo/charts
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts", apiPrefix))

	// GET /api/:repo/charts?offset=10&limit=5
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts?offset=10&limit=5", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts", apiPrefix))

	// GET /api/:repo/charts?offset=-1
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts?offset=-1&limit=5", apiPrefix), nil, "")
	suite.Equal(400, res.Status(), fmt.Sprintf("400 GET %s/charts?offset=-1", apiPrefix))

	// GET /api/:repo/charts?limit=0
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts?offset=-1&limit=5", apiPrefix), nil, "")
	suite.Equal(400, res.Status(), fmt.Sprintf("400 GET %s/charts?limit=0", apiPrefix))

	// GET /api/:repo/charts/:name
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/fakechart", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("404 GET %s/charts/fakechart", apiPrefix))

	// HEAD /api/:repo/charts/:name
	res = suite.doRequest(stype, "HEAD", fmt.Sprintf("%s/charts/mychart", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 HEAD %s/charts/mychart", apiPrefix))

	res = suite.doRequest(stype, "HEAD", fmt.Sprintf("%s/charts/fakechart", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("404 HEAD %s/charts/fakechart", apiPrefix))

	// GET /api/:repo/charts/:name/:version
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart/0.1.0", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart/0.1.0", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart/latest", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart/latest", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart/0.1.1", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart/0.1.1", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/fakechart/0.1.0", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("200 GET %s/charts/fakechart/0.1.0", apiPrefix))

	// GET /api/:repo/charts/:name/:version/templates
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart/0.1.0/templates", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart/0.1.0/templates", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart/latest/templates", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart/latest/templates", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart/0.1.1/templates", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("404 GET %s/charts/mychart/0.1.1/templates", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/fakechart/0.1.0/templates", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("404 GET %s/charts/fakechart/0.1.0/templates", apiPrefix))

	// GET /api/:repo/charts/:name/:version/values
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart/0.1.0/values", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart/0.1.0/values", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart/latest/values", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/charts/mychart/latest/values", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/mychart/0.1.1/values", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("404 GET %s/charts/mychart/0.1.1/values", apiPrefix))

	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/charts/fakechart/0.1.0/values", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("404 GET %s/charts/fakechart/0.1.0/values", apiPrefix))

	// HEAD /api/:repo/charts/:name/:version
	res = suite.doRequest(stype, "HEAD", fmt.Sprintf("%s/charts/mychart/0.1.0", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 HEAD %s/charts/mychart/0.1.0", apiPrefix))

	res = suite.doRequest(stype, "HEAD", fmt.Sprintf("%s/charts/mychart/latest", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 HEAD %s/charts/mychart/latest", apiPrefix))

	res = suite.doRequest(stype, "HEAD", fmt.Sprintf("%s/charts/mychart/0.1.1", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("200 HEAD %s/charts/mychart/0.1.1", apiPrefix))

	res = suite.doRequest(stype, "HEAD", fmt.Sprintf("%s/charts/fakechart/0.1.0", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("200 HEAD %s/charts/fakechart/0.1.0", apiPrefix))

	// DELETE /api/:repo/charts/:name/:version
	res = suite.doRequest(stype, "DELETE", fmt.Sprintf("%s/charts/mychart/0.1.0", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 DELETE %s/charts/mychart/0.1.0", apiPrefix))

	res = suite.doRequest(stype, "DELETE", fmt.Sprintf("%s/charts/mychart/0.1.0", apiPrefix), nil, "")
	suite.Equal(404, res.Status(), fmt.Sprintf("200 DELETE %s/charts/mychart/0.1.0", apiPrefix))

	// GET /:repo/index.yaml (after delete)
	res = suite.doRequest(stype, "GET", fmt.Sprintf("%s/index.yaml", repoPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 GET %s/index.yaml", repoPrefix))

	// POST /api/:repo/charts
	body := bytes.NewBuffer([]byte{})
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/charts", apiPrefix), body, "")
	suite.Equal(400, res.Status(), fmt.Sprintf("400 POST %s/charts", apiPrefix))

	// POST /api/:repo/prov
	body = bytes.NewBuffer([]byte{})
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/prov", apiPrefix), body, "")
	suite.Equal(500, res.Status(), fmt.Sprintf("500 POST %s/prov", apiPrefix))

	// POST /api/:repo/charts
	content, err := ioutil.ReadFile(testTarballPath)
	suite.Nil(err, "no error opening test tarball")

	body = bytes.NewBuffer(content)
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/charts", apiPrefix), body, "")
	suite.Equal(201, res.Status(), fmt.Sprintf("201 POST %s/charts", apiPrefix))

	body = bytes.NewBuffer(content)
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/charts", apiPrefix), body, "")
	suite.Equal(409, res.Status(), fmt.Sprintf("409 POST %s/charts", apiPrefix))

	// with force, still 409
	body = bytes.NewBuffer(content)
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/charts?force", apiPrefix), body, "")
	suite.Equal(409, res.Status(), fmt.Sprintf("409 POST %s/charts?force", apiPrefix))

	// POST /api/:repo/prov
	content, err = ioutil.ReadFile(testProvfilePath)
	suite.Nil(err, "no error opening test provenance file")

	body = bytes.NewBuffer(content)
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/prov", apiPrefix), body, "")
	suite.Equal(201, res.Status(), fmt.Sprintf("201 POST %s/prov", apiPrefix))

	body = bytes.NewBuffer(content)
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/prov", apiPrefix), body, "")
	suite.Equal(409, res.Status(), fmt.Sprintf("409 POST %s/prov", apiPrefix))

	// with force, still 409
	body = bytes.NewBuffer(content)
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/prov?force", apiPrefix), body, "")
	suite.Equal(409, res.Status(), fmt.Sprintf("409 POST %s/prov?force", apiPrefix))

	// Clear test repo to allow uploading again
	res = suite.doRequest(stype, "DELETE", fmt.Sprintf("%s/charts/mychart/0.1.0", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 DELETE %s/charts/mychart/0.1.0", apiPrefix))

	// Create form file with chart=@mychart-0.1.0.tgz
	buf, w := suite.getBodyWithMultipartFormFiles([]string{"chart"}, []string{testTarballPath})
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/charts", apiPrefix), buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), fmt.Sprintf("201 POST %s/charts", apiPrefix))

	// Create form file with prov=@mychart-0.1.0.tgz.prov
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"prov"}, []string{testProvfilePath})
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/charts", apiPrefix), buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), fmt.Sprintf("201 POST %s/charts", apiPrefix))

	// Clear test repo to allow uploading again
	res = suite.doRequest(stype, "DELETE", fmt.Sprintf("%s/charts/mychart/0.1.0", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 DELETE %s/charts/mychart/0.1.0", apiPrefix))

	// Create form file with chart=@mychart-0.1.0.tgz and prov=@mychart-0.1.0.tgz.prov
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart", "prov"}, []string{testTarballPath, testProvfilePath})
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/charts", apiPrefix), buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), fmt.Sprintf("201 POST %s/charts", apiPrefix))

	// Clear test repo to allow uploading again
	res = suite.doRequest(stype, "DELETE", fmt.Sprintf("%s/charts/mychart/0.1.0", apiPrefix), nil, "")
	suite.Equal(200, res.Status(), fmt.Sprintf("200 DELETE %s/charts/mychart/0.1.0", apiPrefix))

	// Create form file with unknown=@mychart-0.1.0.tgz, which should fail because the server doesn't know about the unknown field
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"unknown"}, []string{testTarballPath})
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/charts", apiPrefix), buf, w.FormDataContentType())
	suite.Equal(400, res.Status(), fmt.Sprintf("400 POST %s/charts", apiPrefix))

	// Create form file with chart=@mychart-0.1.0.tgz
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart"}, []string{testTarballPath})
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/charts", apiPrefix), buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), fmt.Sprintf("201 POST %s/charts", apiPrefix))

	// Create form file with chart=@mychart-0.1.0.tgz, which should fail because it is already there
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart"}, []string{testTarballPath})
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/charts", apiPrefix), buf, w.FormDataContentType())
	suite.Equal(409, res.Status(), fmt.Sprintf("409 POST %s/charts", apiPrefix))

	// with force, still 409
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart"}, []string{testTarballPath})
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/charts?force", apiPrefix), buf, w.FormDataContentType())
	suite.Equal(409, res.Status(), fmt.Sprintf("409 POST %s/charts?force", apiPrefix))

	// Create form file with chart=@mychart-0.1.0.tgz.prov, which should fail because it is not a valid chart package
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart"}, []string{testProvfilePath})
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/charts", apiPrefix), buf, w.FormDataContentType())
	suite.Equal(400, res.Status(), fmt.Sprintf("400 POST %s/charts", apiPrefix))

	// Create form file with prov=@mychart-0.1.0.tgz, which should fail because it is not a valid provenance file
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"prov"}, []string{testTarballPath})
	res = suite.doRequest(stype, "POST", fmt.Sprintf("%s/charts", apiPrefix), buf, w.FormDataContentType())
	suite.Equal(400, res.Status(), fmt.Sprintf("400 POST %s/charts", apiPrefix))

}

func (suite *MultiTenantServerTestSuite) getBodyWithMultipartFormFiles(fields []string, filenames []string) (io.Reader, *multipart.Writer) {
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)
	for i := range fields {
		fw, err := w.CreateFormFile(fields[i], filenames[i])
		suite.Nil(err, "no error creating form file")
		fd, err := os.Open(filenames[i])
		suite.Nil(err, "no error opening test file")
		defer fd.Close()
		_, err = io.Copy(fw, fd)
		suite.Nil(err, "no error copying test file to form file")
	}
	w.Close()
	return buf, w
}

func TestMultiTenantServerTestSuite(t *testing.T) {
	suite.Run(t, new(MultiTenantServerTestSuite))
}
