package chartmuseum

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	pathutil "path"
	"testing"
	"time"

	"github.com/kubernetes-helm/chartmuseum/pkg/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

var testTarballPath = "../../testdata/charts/mychart/mychart-0.1.0.tgz"
var testProvfilePath = "../../testdata/charts/mychart/mychart-0.1.0.tgz.prov"

type ServerTestSuite struct {
	suite.Suite
	Server               *Server
	DisabledAPIServer    *Server
	BrokenServer         *Server
	OverwriteServer      *Server
	TempDirectory        string
	BrokenTempDirectory  string
	TestTarballFilename  string
	TestProvfileFilename string
}

func (suite *ServerTestSuite) doRequest(stype string, method string, urlStr string, body io.Reader, contentType string) gin.ResponseWriter {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request, _ = http.NewRequest(method, urlStr, body)
	if contentType != "" {
		c.Request.Header.Set("Content-Type", contentType)
	}

	switch stype {
	case "normal":
		c.Request.SetBasicAuth("user", "pass")
		suite.Server.Router.HandleContext(c)
	case "broken":
		suite.BrokenServer.Router.HandleContext(c)
	case "disabled":
		suite.DisabledAPIServer.Router.HandleContext(c)
	case "overwrite":
		suite.OverwriteServer.Router.HandleContext(c)
	}

	return c.Writer
}

func (suite *ServerTestSuite) SetupSuite() {
	srcFileTarball, err := os.Open(testTarballPath)
	suite.Nil(err, "no error opening test tarball")
	defer srcFileTarball.Close()

	srcFileProvfile, err := os.Open(testTarballPath)
	suite.Nil(err, "no error opening test provfile")
	defer srcFileProvfile.Close()

	timestamp := time.Now().Format("20060102150405")
	suite.TempDirectory = fmt.Sprintf("../../.test/chartmuseum-server/%s", timestamp)

	backend := storage.Backend(storage.NewLocalFilesystemBackend(suite.TempDirectory))

	server, err := NewServer(ServerOptions{backend, false, false, true, false, false, "", "", "", "", "", "", ""})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new server, logJson=false, debug=false, disabled=false, overwrite=false")

	server, err = NewServer(ServerOptions{backend, true, true, true, false, false, "", "", "", "", "", "", ""})
	suite.NotNil(server)
	suite.Nil(err, "no error creating new server, logJson=true, debug=true, disabled=false, overwrite=false")

	server, err = NewServer(ServerOptions{backend, false, true, true, false, false, "", "", "", "user", "pass", "chart", "prov"})
	suite.Nil(err, "no error creating new server, logJson=false, debug=true, disabled=false, overwrite=false")

	suite.Server = server

	disabledAPIServer, err := NewServer(ServerOptions{backend, false, true, false, false, false, "", "", "", "", "", "", ""})
	suite.Nil(err, "no error creating new server, logJson=false, debug=true, disabled=true, overwrite=false")

	suite.DisabledAPIServer = disabledAPIServer

	overwriteServer, err := NewServer(ServerOptions{backend, false, true, true, true, false, "", "", "", "", "", "chart", "prov"})
	suite.Nil(err, "no error creating new server, logJson=false, debug=true, disabled=false, overwrite=true")

	suite.OverwriteServer = overwriteServer

	suite.TestTarballFilename = pathutil.Join(suite.TempDirectory, "mychart-0.1.0.tgz")
	destFileTarball, err := os.Create(suite.TestTarballFilename)
	suite.Nil(err, "no error creating new tarball in temp dir")
	defer destFileTarball.Close()

	_, err = io.Copy(destFileTarball, srcFileTarball)
	suite.Nil(err, "no error copying test testball to temp tarball")

	err = destFileTarball.Sync()
	suite.Nil(err, "no error syncing temp tarball")

	suite.TestProvfileFilename = pathutil.Join(suite.TempDirectory, "mychart-0.1.0.tgz.prov")
	destFileProvfile, err := os.Create(suite.TestProvfileFilename)
	suite.Nil(err, "no error creating new provenance file in temp dir")
	defer destFileProvfile.Close()

	_, err = io.Copy(destFileProvfile, srcFileProvfile)
	suite.Nil(err, "no error copying test provenance file to temp tarball")

	err = destFileProvfile.Sync()
	suite.Nil(err, "no error syncing temp provenance file")

	suite.BrokenTempDirectory = fmt.Sprintf("../../.test/chartmuseum-server/%s-broken", timestamp)
	defer os.RemoveAll(suite.BrokenTempDirectory)

	brokenBackend := storage.Backend(storage.NewLocalFilesystemBackend(suite.BrokenTempDirectory))
	brokenServer, err := NewServer(ServerOptions{brokenBackend, false, true, true, false, false, "", "", "", "", "", "", ""})
	suite.Nil(err, "no error creating new server, logJson=false, debug=true, disabled=false, overwrite=false")

	suite.BrokenServer = brokenServer
}

func (suite *ServerTestSuite) TearDownSuite() {
	err := os.RemoveAll(suite.TempDirectory)
	suite.Nil(err, "no error deleting temp directory for local storage")
}

func (suite *ServerTestSuite) TestRegenerateRepositoryIndex() {
	objects, err := suite.Server.fetchChartsInStorage("test")
	diff := storage.GetObjectSliceDiff(suite.Server.StorageCache, objects)
	_, err = suite.Server.regenerateRepositoryIndexWorker(diff, objects, "test")
	suite.Nil(err, "no error regenerating repo index")

	newtime := time.Now().Add(1 * time.Hour)
	err = os.Chtimes(suite.TestTarballFilename, newtime, newtime)
	suite.Nil(err, "no error changing modtime on temp file")

	objects, err = suite.Server.fetchChartsInStorage("test")
	diff = storage.GetObjectSliceDiff(suite.Server.StorageCache, objects)
	_, err = suite.Server.regenerateRepositoryIndexWorker(diff, objects, "test")
	suite.Nil(err, "no error regenerating repo index with tarball updated")

	brokenTarballFilename := pathutil.Join(suite.TempDirectory, "brokenchart.tgz")
	destFile, err := os.Create(brokenTarballFilename)
	suite.Nil(err, "no error creating new broken tarball in temp dir")
	defer destFile.Close()
	objects, err = suite.Server.fetchChartsInStorage("test")
	diff = storage.GetObjectSliceDiff(suite.Server.StorageCache, objects)
	_, err = suite.Server.regenerateRepositoryIndexWorker(diff, objects, "test")
	suite.Nil(err, "error not returned with broken tarball added")

	err = os.Chtimes(brokenTarballFilename, newtime, newtime)
	suite.Nil(err, "no error changing modtime on broken tarball")
	objects, err = suite.Server.fetchChartsInStorage("test")
	diff = storage.GetObjectSliceDiff(suite.Server.StorageCache, objects)
	_, err = suite.Server.regenerateRepositoryIndexWorker(diff, objects, "test")
	suite.Nil(err, "error not returned with broken tarball updated")

	err = os.Remove(brokenTarballFilename)
	suite.Nil(err, "no error removing broken tarball")
	objects, err = suite.Server.fetchChartsInStorage("test")
	diff = storage.GetObjectSliceDiff(suite.Server.StorageCache, objects)
	_, err = suite.Server.regenerateRepositoryIndexWorker(diff, objects, "test")
	suite.Nil(err, "error not returned with broken tarball removed")
}

func (suite *ServerTestSuite) TestRoutes() {
	var body io.Reader
	var res gin.ResponseWriter

	// GET /charts/<filename>
	res = suite.doRequest("normal", "GET", "/charts/mychart-0.1.0.tgz", nil, "")
	suite.Equal(200, res.Status(), "200 GET /charts/mychart-0.1.0.tgz")

	res = suite.doRequest("normal", "GET", "/charts/mychart-0.1.0.tgz.prov", nil, "")
	suite.Equal(200, res.Status(), "200 GET /charts/mychart-0.1.0.tgz.prov")

	res = suite.doRequest("normal", "GET", "/charts/fakechart-0.1.0.tgz", nil, "")
	suite.Equal(404, res.Status(), "404 GET /charts/fakechart-0.1.0.tgz")

	res = suite.doRequest("normal", "GET", "/charts/fakechart-0.1.0.tgz.prov", nil, "")
	suite.Equal(404, res.Status(), "404 GET /charts/fakechart-0.1.0.tgz.prov")

	res = suite.doRequest("normal", "GET", "/charts/fakechart-0.1.0.bad", nil, "")
	suite.Equal(500, res.Status(), "500 GET /charts/fakechart-0.1.0.bad")

	// GET /api/charts
	res = suite.doRequest("normal", "GET", "/api/charts", nil, "")
	suite.Equal(200, res.Status(), "200 GET /api/charts")

	res = suite.doRequest("broken", "GET", "/api/charts", nil, "")
	suite.Equal(500, res.Status(), "500 GET /api/charts")

	// GET /api/charts/<chart>
	res = suite.doRequest("normal", "GET", "/api/charts/mychart", nil, "")
	suite.Equal(200, res.Status(), "200 GET /api/charts/mychart")

	res = suite.doRequest("normal", "GET", "/api/charts/fakechart", nil, "")
	suite.Equal(404, res.Status(), "404 GET /api/charts/fakechart")

	res = suite.doRequest("broken", "GET", "/api/charts/mychart", nil, "")
	suite.Equal(500, res.Status(), "500 GET /api/charts/mychart")

	// GET /api/charts/<chart>/<version>
	res = suite.doRequest("normal", "GET", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(200, res.Status(), "200 GET /api/charts/mychart/0.1.0")

	res = suite.doRequest("normal", "GET", "/api/charts/mychart/latest", nil, "")
	suite.Equal(200, res.Status(), "200 GET /api/charts/mychart/latest")

	res = suite.doRequest("normal", "GET", "/api/charts/mychart/0.0.0", nil, "")
	suite.Equal(404, res.Status(), "404 GET /api/charts/mychart/0.0.0")

	res = suite.doRequest("normal", "GET", "/api/charts/fakechart/0.1.0", nil, "")
	suite.Equal(404, res.Status(), "404 GET /api/charts/fakechart/0.1.0")

	res = suite.doRequest("broken", "GET", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(500, res.Status(), "500 GET /api/charts/mychart/0.1.0")

	// DELETE /api/charts/<chart>/<version>
	res = suite.doRequest("normal", "DELETE", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(200, res.Status(), "200 DELETE /api/charts/mychart/0.1.0")

	res = suite.doRequest("normal", "DELETE", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(404, res.Status(), "404 DELETE /api/charts/mychart/0.1.0")

	// GET /index.yaml
	res = suite.doRequest("normal", "GET", "/index.yaml", nil, "")
	suite.Equal(200, res.Status(), "200 GET /index.yaml")

	res = suite.doRequest("broken", "GET", "/index.yaml", nil, "")
	suite.Equal(500, res.Status(), "500 GET /index.yaml")

	// POST /api/charts
	body = bytes.NewBuffer([]byte{})
	res = suite.doRequest("normal", "POST", "/api/charts", body, "")
	suite.Equal(500, res.Status(), "500 POST /api/charts")

	// POST /api/prov
	body = bytes.NewBuffer([]byte{})
	res = suite.doRequest("normal", "POST", "/api/prov", body, "")
	suite.Equal(500, res.Status(), "500 POST /api/prov")

	// POST /api/charts
	content, err := ioutil.ReadFile(testTarballPath)
	suite.Nil(err, "no error opening test tarball")

	body = bytes.NewBuffer(content)
	res = suite.doRequest("normal", "POST", "/api/charts", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/charts")

	body = bytes.NewBuffer(content)
	res = suite.doRequest("normal", "POST", "/api/charts", body, "")
	suite.Equal(500, res.Status(), "500 POST /api/charts")

	// POST /api/prov
	content, err = ioutil.ReadFile(testProvfilePath)
	suite.Nil(err, "no error opening test provenance file")

	body = bytes.NewBuffer(content)
	res = suite.doRequest("normal", "POST", "/api/prov", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/prov")

	body = bytes.NewBuffer(content)
	res = suite.doRequest("normal", "POST", "/api/prov", body, "")
	suite.Equal(500, res.Status(), "500 POST /api/prov")

	// Test that all /api routes disabled if EnableAPI=false
	res = suite.doRequest("disabled", "GET", "/api/charts", nil, "")
	suite.Equal(404, res.Status(), "404 GET /api/charts")

	res = suite.doRequest("disabled", "GET", "/api/charts/mychart", nil, "")
	suite.Equal(404, res.Status(), "404 GET /api/charts")

	res = suite.doRequest("disabled", "GET", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(404, res.Status(), "404 GET /api/charts")

	body = bytes.NewBuffer([]byte{})
	res = suite.doRequest("disabled", "POST", "/api/charts", body, "")
	suite.Equal(404, res.Status(), "404 POST /api/charts")

	body = bytes.NewBuffer([]byte{})
	res = suite.doRequest("disabled", "POST", "/api/prov", body, "")
	suite.Equal(404, res.Status(), "404 POST /api/prov")

	res = suite.doRequest("disabled", "DELETE", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(404, res.Status(), "404 DELETE /api/charts/mychart/0.1.0")

	// Clear test repo to allow uploading again
	res = suite.doRequest("normal", "DELETE", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(200, res.Status(), "200 DELETE /api/charts/mychart/0.1.0")

	// Create form file with chart=@mychart-0.1.0.tgz
	buf, w := suite.getBodyWithMultipartFormFiles([]string{"chart"}, []string{testTarballPath})
	res = suite.doRequest("normal", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), "201 POST /api/charts")

	// Create form file with prov=@mychart-0.1.0.tgz.prov
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"prov"}, []string{testProvfilePath})
	res = suite.doRequest("normal", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), "201 POST /api/charts")

	// Clear test repo to allow uploading again
	res = suite.doRequest("normal", "DELETE", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(200, res.Status(), "200 DELETE /api/charts/mychart/0.1.0")

	// Create form file with chart=@mychart-0.1.0.tgz and prov=@mychart-0.1.0.tgz.prov
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart", "prov"}, []string{testTarballPath, testProvfilePath})
	res = suite.doRequest("normal", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), "201 POST /api/charts")

	// Clear test repo to allow uploading again
	res = suite.doRequest("normal", "DELETE", "/api/charts/mychart/0.1.0", nil, "")
	suite.Equal(200, res.Status(), "200 DELETE /api/charts/mychart/0.1.0")

	// Create form file with unknown=@mychart-0.1.0.tgz, which should fail because the server doesn't know about the unknown field
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"unknown"}, []string{testTarballPath})
	res = suite.doRequest("normal", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(400, res.Status(), "400 POST /api/charts")

	// Create form file with chart=@mychart-0.1.0.tgz
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart"}, []string{testTarballPath})
	res = suite.doRequest("normal", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), "201 POST /api/charts")

	// Create form file with chart=@mychart-0.1.0.tgz, which should fail because it is already there
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart"}, []string{testTarballPath})
	res = suite.doRequest("normal", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(409, res.Status(), "409 POST /api/charts")

	// Create form file with chart=@mychart-0.1.0.tgz.prov, which should fail because it is not a valid chart package
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart"}, []string{testProvfilePath})
	res = suite.doRequest("normal", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(400, res.Status(), "400 POST /api/charts")

	// Create form file with prov=@mychart-0.1.0.tgz, which should fail because it is not a valid provenance file
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"prov"}, []string{testTarballPath})
	res = suite.doRequest("normal", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(400, res.Status(), "400 POST /api/charts")

	// Check if files can be overwritten
	content, err = ioutil.ReadFile(testTarballPath)
	suite.Nil(err, "no error opening test tarball")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("overwrite", "POST", "/api/charts", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/charts")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("overwrite", "POST", "/api/charts", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/charts")

	content, err = ioutil.ReadFile(testProvfilePath)
	suite.Nil(err, "no error opening test provenance file")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("overwrite", "POST", "/api/prov", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/prov")
	body = bytes.NewBuffer(content)
	res = suite.doRequest("overwrite", "POST", "/api/prov", body, "")
	suite.Equal(201, res.Status(), "201 POST /api/prov")

	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart", "prov"}, []string{testTarballPath, testProvfilePath})
	res = suite.doRequest("overwrite", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), "201 POST /api/charts")
	buf, w = suite.getBodyWithMultipartFormFiles([]string{"chart", "prov"}, []string{testTarballPath, testProvfilePath})
	res = suite.doRequest("overwrite", "POST", "/api/charts", buf, w.FormDataContentType())
	suite.Equal(201, res.Status(), "201 POST /api/charts")
}

func (suite *ServerTestSuite) getBodyWithMultipartFormFiles(fields []string, filenames []string) (io.Reader, *multipart.Writer) {
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

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}

func TestMapURLWithParamsBackToRouteTemplate(t *testing.T) {
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
		got := mapURLWithParamsBackToRouteTemplate(tt.ctx)
		if got != tt.expect {
			t.Errorf("expected %s, got %s", tt.expect, got)
		}
	}
}
