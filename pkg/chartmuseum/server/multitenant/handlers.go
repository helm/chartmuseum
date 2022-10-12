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
	"net/http"
	"os"
	pathutil "path"
	"strconv"
	"time"

	cm_storage "github.com/chartmuseum/storage"

	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "helm.sh/chartmuseum/pkg/repo"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	helm_repo "helm.sh/helm/v3/pkg/repo"

	"github.com/gin-gonic/gin"

	"go.uber.org/zap"
)

var (
	objectSavedResponse   = gin.H{"saved": true}
	objectDeletedResponse = gin.H{"deleted": true}
	healthCheckResponse   = gin.H{"healthy": true}
	welcomePageHTML       = []byte(`<!DOCTYPE html>
<html>
<head>
<title>Welcome to ChartMuseum!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to ChartMuseum!</h1>
<p>If you see this page, the ChartMuseum web server is successfully installed and
working.</p>

<p>For online documentation and support please refer to the
<a href="https://github.com/helm/chartmuseum">GitHub project</a>.<br/>

<p><em>Thank you for using ChartMuseum.</em></p>
</body>
</html>
`)
)

type (
	HTTPError struct {
		Status  int
		Message string
	}
)

type (
	chartOrProvenanceFile struct {
		filename string
		content  []byte
		field    string // file was extracted from this form field
	}
	filenameFromContentFn func([]byte) (string, error)
)

func (server *MultiTenantServer) getWelcomePageHandler(c *gin.Context) {
	if server.WebTemplatePath != "" {
		// Check if template file exists, otherwise return default welcome page
		templateFilesExist := server.CheckTemplateFilesExist(server.WebTemplatePath, server.Logger)
		if templateFilesExist {
			c.HTML(http.StatusOK, "index.html", nil)
		} else {
			server.Logger.Warnf("No template files found in %s, fallback to default welcome page", server.WebTemplatePath)
			c.Data(http.StatusOK, "text/html", welcomePageHTML)
		}
	} else {
		c.Data(http.StatusOK, "text/html", welcomePageHTML)
	}
}

func (server *MultiTenantServer) getStaticFilesHandler(c *gin.Context) {
	staticFolder := fmt.Sprintf("%s/static", server.WebTemplatePath)
	if _, err := os.Stat(staticFolder); !os.IsNotExist(err) {
		c.File(fmt.Sprintf("%s%s", server.WebTemplatePath, c.Request.URL.Path))
	}
}

func (server *MultiTenantServer) getInfoHandler(c *gin.Context) {
	versionResponse := gin.H{"version": server.Version}
	c.JSON(200, versionResponse)
}

func (server *MultiTenantServer) getHealthCheckHandler(c *gin.Context) {
	c.JSON(200, healthCheckResponse)
}

func (server *MultiTenantServer) getIndexFileRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	log := server.Logger.ContextLoggingFn(c)
	indexFile, err := server.getIndexFile(log, repo)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	indexFile.IndexLock.RLock()
	defer indexFile.IndexLock.RUnlock()
	c.Data(200, indexFileContentType, indexFile.Raw)
}

func (server *MultiTenantServer) headIndexFileRequestHandler(c *gin.Context) {
	c.Status(200)
}

func (server *MultiTenantServer) getArtifactHubFileRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	log := server.Logger.ContextLoggingFn(c)
	artifactHubFile, err := server.getArtifactHubYml(log, repo)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}

	c.Data(200, artifactHubFileContentType, artifactHubFile)
}

func (server *MultiTenantServer) getStorageObjectRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	filename := c.Param("filename")
	log := server.Logger.ContextLoggingFn(c)
	storageObject, err := server.getStorageObject(log, repo, filename)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	c.Data(200, storageObject.ContentType, storageObject.Content)
}
func (server *MultiTenantServer) getStorageObjectTemplateRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	name := c.Param("name")
	version := c.Param("version")

	log := server.Logger.ContextLoggingFn(c)

	fileName, err := server.getChartFileName(log, repo, name, version)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Message})
		return
	}

	storageObject, err := server.getStorageObject(log, repo, fileName)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	chrt, err1 := loader.LoadArchive(bytes.NewReader(storageObject.Content))
	if err1 != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err1})
		return
	}
	c.JSON(200, map[string]interface{}{
		"templates": chrt.Templates,
		"values":    chrt.Values,
	})
}

func (server *MultiTenantServer) getStorageObjectValuesRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	name := c.Param("name")
	version := c.Param("version")

	log := server.Logger.ContextLoggingFn(c)

	fileName, err := server.getChartFileName(log, repo, name, version)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Message})
		return
	}

	storageObject, err := server.getStorageObject(log, repo, fileName)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	chrt, err1 := loader.LoadArchive(bytes.NewReader(storageObject.Content))
	if err1 != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err1})
		return
	}
	var data []byte
	for _, file := range chrt.Raw {
		if file.Name == "values.yaml" {
			data = file.Data
			break
		}
	}
	if data == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "values.yaml not found"})
		return
	}
	c.Data(200, "application/yaml", data)
}
func (server *MultiTenantServer) getAllChartsRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	offset := 0
	offsetString, offsetExists := c.GetQuery("offset")
	if offsetExists {
		var convErr error
		offset, convErr = strconv.Atoi(offsetString)
		if convErr != nil || offset < 0 {
			c.JSON(400, gin.H{"error": "offset is not a valid non-negative integer"})
			return
		}
	}

	limit := -1
	limitString, limitExists := c.GetQuery("limit")
	if limitExists {
		var convErr error
		limit, convErr = strconv.Atoi(limitString)
		if convErr != nil || limit <= 0 {
			c.JSON(400, gin.H{"error": "limit is not a valid positive integer"})
			return
		}
	}

	log := server.Logger.ContextLoggingFn(c)
	allCharts, err := server.getAllCharts(log, repo, offset, limit)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	c.JSON(200, allCharts)
}

func (server *MultiTenantServer) getChartRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	name := c.Param("name")
	log := server.Logger.ContextLoggingFn(c)
	chart, err := server.getChart(log, repo, name)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	c.JSON(200, chart)
}

func (server *MultiTenantServer) headChartRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	name := c.Param("name")
	log := server.Logger.ContextLoggingFn(c)
	_, err := server.getChart(log, repo, name)
	if err != nil {
		c.Status(err.Status)
		return
	}
	c.Status(200)
}

func (server *MultiTenantServer) getChartVersionRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	name := c.Param("name")
	version := c.Param("version")
	log := server.Logger.ContextLoggingFn(c)
	chartVersion, err := server.getChartVersion(log, repo, name, version)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	c.JSON(200, chartVersion)
}

func (server *MultiTenantServer) headChartVersionRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	name := c.Param("name")
	version := c.Param("version")
	log := server.Logger.ContextLoggingFn(c)
	_, err := server.getChartVersion(log, repo, name, version)
	if err != nil {
		c.Status(err.Status)
		return
	}
	c.Status(200)
}

func (server *MultiTenantServer) deleteChartVersionRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	name := c.Param("name")
	version := c.Param("version")
	log := server.Logger.ContextLoggingFn(c)
	err := server.deleteChartVersion(log, repo, name, version)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}

	server.emitEvent(c, repo, deleteChart, &helm_repo.ChartVersion{
		Metadata: &chart.Metadata{
			Name:    name,
			Version: version,
		},
		// Since we only need name and version to delete the chart version from index
		// left the others fields to be default
	})
	c.JSON(200, objectDeletedResponse)
}

func (server *MultiTenantServer) postRequestHandler(c *gin.Context) {
	if c.ContentType() == "multipart/form-data" {
		server.postPackageAndProvenanceRequestHandler(c) // new route handling form-based chart and/or prov files
	} else {
		server.postPackageRequestHandler(c) // classic binary data, chart package only route
	}
}

func (server *MultiTenantServer) postPackageRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	content, getContentErr := c.GetRawData()
	if getContentErr != nil {
		if len(c.Errors) > 0 {
			return // this is a "request too large"
		}
		c.JSON(500, gin.H{"error": fmt.Sprintf("%s", getContentErr)})
		return
	}
	log := server.Logger.ContextLoggingFn(c)
	_, force := c.GetQuery("force")
	action := addChart
	filename, err := server.uploadChartPackage(log, repo, content, force)
	if err != nil {
		// here should check both err.Status and err.Message
		// The http.StatusConflict status means the chart is existed but overwrite is not sed OR chart is existed and overwrite is set
		// err.Status == http.StatusConflict only denotes for chart is existed now.
		if err.Status == http.StatusConflict {
			if err.Message != "" {
				c.JSON(err.Status, gin.H{"error": err.Message})
				return
			}
			action = updateChart
		} else {
			c.JSON(err.Status, gin.H{"error": err.Message})
			return
		}
	}

	chart, chartErr := cm_repo.ChartVersionFromStorageObject(cm_storage.Object{
		Path:         pathutil.Join(repo, filename),
		Content:      content,
		LastModified: time.Now()})
	if chartErr != nil {
		log(cm_logger.ErrorLevel, "cannot get chart from content", zap.Error(chartErr), zap.Binary("content", content))
	}
	server.emitEvent(c, repo, action, chart)

	c.JSON(201, objectSavedResponse)
}

// TODO: whether need update cache
func (server *MultiTenantServer) postProvenanceFileRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	content, getContentErr := c.GetRawData()
	if getContentErr != nil {
		if len(c.Errors) > 0 {
			return // this is a "request too large"
		}
		c.JSON(500, gin.H{"error": fmt.Sprintf("%s", getContentErr)})
		return
	}
	log := server.Logger.ContextLoggingFn(c)
	_, force := c.GetQuery("force")
	err := server.uploadProvenanceFile(log, repo, content, force)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	c.JSON(201, objectSavedResponse)
}

func (server *MultiTenantServer) postPackageAndProvenanceRequestHandler(c *gin.Context) {
	log := server.Logger.ContextLoggingFn(&gin.Context{})
	repo := c.Param("repo")
	_, force := c.GetQuery("force")
	var chartContent []byte
	var path string
	// action used to determine what operation to emit
	action := addChart
	cpFiles, status, err := server.getChartAndProvFiles(c.Request, repo, force)
	if err != nil {
		c.JSON(status, gin.H{"error": fmt.Sprintf("%s", err)})
		return
	}
	switch status {
	case http.StatusOK:
	case http.StatusConflict:
		if !server.AllowOverwrite && (!server.AllowForceOverwrite || !force) {
			c.JSON(status, gin.H{"error": fmt.Sprintf("%s", fmt.Errorf("chart already exists"))}) // conflict
			return
		}
		log(cm_logger.DebugLevel, "chart already exists, but overwrite is allowed", zap.String("repo", repo))
		// update chart if chart already exists and overwrite is allowed
		action = updateChart
	default:
		c.JSON(status, gin.H{"error": fmt.Sprintf("%s", err)})
		return
	}

	if len(cpFiles) == 0 {
		if len(c.Errors) > 0 {
			return // this is a "request too large"
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf(
			"no package or provenance file found in form fields %s and %s",
			server.ChartPostFormFieldName, server.ProvPostFormFieldName),
		})
		return
	}

	// At this point input is presumed valid, we now proceed to store it
	// Undo transaction if there is an error
	var storedFiles []*chartOrProvenanceFile
	for _, ppf := range cpFiles {
		server.Logger.Debugc(c, "Adding file to storage (form field)",
			"filename", ppf.filename,
			"field", ppf.field,
		)
		err := server.StorageBackend.PutObject(pathutil.Join(repo, ppf.filename), ppf.content)
		if err == nil {
			storedFiles = append(storedFiles, ppf)
		} else {
			// Clean up what's already been saved
			for _, ppf := range storedFiles {
				server.StorageBackend.DeleteObject(ppf.filename)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("%s", err)})
			return
		}
		if ppf.field == defaultFormField {
			// find the content of chart
			chartContent = ppf.content
			path = pathutil.Join(repo, ppf.filename)
		}
	}

	chart, chartErr := cm_repo.ChartVersionFromStorageObject(cm_storage.Object{
		Path:         path,
		Content:      chartContent,
		LastModified: time.Now()})
	if chartErr != nil {
		log(cm_logger.ErrorLevel, "cannot get chart from content", zap.Error(err), zap.Binary("content", chartContent))
	}

	server.emitEvent(c, repo, action, chart)

	c.JSON(http.StatusCreated, objectSavedResponse)
}

func (server *MultiTenantServer) getChartAndProvFiles(req *http.Request, repo string, force bool) (map[string]*chartOrProvenanceFile, int, error) {
	type fieldFuncPair struct {
		field string
		fn    filenameFromContentFn
	}

	ffp := []fieldFuncPair{
		{defaultFormField, cm_repo.ChartPackageFilenameFromContent},
		{server.ChartPostFormFieldName, cm_repo.ChartPackageFilenameFromContent},
		{defaultProvField, cm_repo.ProvenanceFilenameFromContent},
		{server.ProvPostFormFieldName, cm_repo.ProvenanceFilenameFromContent},
	}

	validReturnStatusCode := http.StatusOK
	cpFiles := make(map[string]*chartOrProvenanceFile)
	for _, ff := range ffp {
		content, err := extractContentFromRequest(req, ff.field)
		if err != nil {
			return nil, http.StatusInternalServerError, err
		}
		if content == nil {
			continue
		}
		filename, err := ff.fn(content)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		if _, ok := cpFiles[filename]; ok {
			continue
		}
		// if the file already exists, we don't need to validate it again
		if validReturnStatusCode == http.StatusConflict {
			cpFiles[filename] = &chartOrProvenanceFile{filename, content, ff.field}
			continue
		}
		// check filename
		if pathutil.Base(filename) != filename {
			return nil, http.StatusBadRequest, fmt.Errorf("%s is improperly formatted", filename) // Name wants to break out of current directory
		}
		// check existence
		status, err := server.validateChartOrProv(repo, filename, force)
		if err != nil {
			return nil, status, err
		}
		// return conflict status code if the file already exists
		if status == http.StatusConflict {
			validReturnStatusCode = status
		}
		cpFiles[filename] = &chartOrProvenanceFile{filename, content, ff.field}
	}

	// validState code can be 200 or 409. Returning 409 means that the chart already exists
	return cpFiles, validReturnStatusCode, nil
}

func extractContentFromRequest(req *http.Request, field string) ([]byte, error) {
	file, header, _ := req.FormFile(field)
	if file == nil || header == nil {
		return nil, nil // field is not present
	}
	buf := bytes.NewBuffer(nil)
	_, err := io.Copy(buf, file)
	if err != nil {
		return nil, err // IO error
	}
	return buf.Bytes(), nil
}

func (server *MultiTenantServer) validateChartOrProv(repo, filename string, force bool) (int, error) {
	var f string
	if repo == "" {
		f = filename
	} else {
		f = repo + "/" + filename
	}
	// conflict does not mean the file is invalid.
	// for example, when overwrite is allowed, it's valid
	// so that the client can decide what to do and here we just return conflict with no error
	if _, err := server.StorageBackend.GetObject(f); err == nil {
		return http.StatusConflict, nil
	}
	return http.StatusOK, nil
}
