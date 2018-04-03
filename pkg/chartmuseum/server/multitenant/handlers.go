package multitenant

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	pathutil "path"

	cm_repo "github.com/kubernetes-helm/chartmuseum/pkg/repo"
)

var (
	objectSavedResponse        = gin.H{"saved": true}
	objectDeletedResponse      = gin.H{"deleted": true}
	healthCheckResponse        = gin.H{"healthy": true}
	welcomePageHTML            = []byte(`<!DOCTYPE html>
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
<a href="https://github.com/kubernetes-helm/chartmuseum">GitHub project</a>.<br/>

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
	packageOrProvenanceFile struct {
		filename string
		content  []byte
		field    string // file was extracted from this form field
	}
	filenameFromContentFn func([]byte) (string, error)
)

func (server *MultiTenantServer) getWelcomePageHandler(c *gin.Context) {
	c.Data(200, "text/html", welcomePageHTML)
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
	c.Data(200, indexFileContentType, indexFile.Raw)
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

func (server *MultiTenantServer) getAllChartsRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	log := server.Logger.ContextLoggingFn(c)
	allCharts, err := server.getAllCharts(log, repo)
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
		c.JSON(500, gin.H{"error": fmt.Sprintf("%s", getContentErr)})
		return
	}
	log := server.Logger.ContextLoggingFn(c)
	err := server.uploadChartPackage(log, repo, content)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	c.JSON(201, objectSavedResponse)
}

func (server *MultiTenantServer) postProvenanceFileRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	content, getContentErr := c.GetRawData()
	if getContentErr != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("%s", getContentErr)})
		return
	}
	log := server.Logger.ContextLoggingFn(c)
	err := server.uploadProvenanceFile(log, repo, content)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	c.JSON(201, objectSavedResponse)
}

// TODO: cleanup, cleanup
func (server *MultiTenantServer) postPackageAndProvenanceRequestHandler(c *gin.Context) {
	repo := c.Param("repo")
	var ppFiles []*packageOrProvenanceFile

	type fieldFuncPair struct {
		field string
		fn    filenameFromContentFn
	}

	ffp := []fieldFuncPair{
		{server.ChartPostFormFieldName, cm_repo.ChartPackageFilenameFromContent},
		{server.ProvPostFormFieldName, cm_repo.ProvenanceFilenameFromContent},
	}

	for _, ff := range ffp {
		ppf, status, err := server.extractAndValidateFormFile(c.Request, repo, ff.field, ff.fn)
		if err != nil {
			c.JSON(status, gin.H{"error": fmt.Sprintf("%s", err)})
			return
		}
		if ppf != nil {
			ppFiles = append(ppFiles, ppf)
		}
	}

	if len(ppFiles) == 0 {
		c.JSON(400, gin.H{"error": fmt.Sprintf(
			"no package or provenance file found in form fields %s and %s",
			server.ChartPostFormFieldName, server.ProvPostFormFieldName),
		})
		return
	}

	// At this point input is presumed valid, we now proceed to store it
	var storedFiles []*packageOrProvenanceFile
	for _, ppf := range ppFiles {
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
			c.JSON(500, gin.H{"error": fmt.Sprintf("%s", err)})
			return
		}
	}
	c.JSON(201, objectSavedResponse)
}

func (server *MultiTenantServer) extractAndValidateFormFile(req *http.Request, repo string, field string, fnFromContent filenameFromContentFn) (*packageOrProvenanceFile, int, error) {
	file, header, _ := req.FormFile(field)
	var ppf *packageOrProvenanceFile
	if file == nil || header == nil {
		return ppf, 200, nil // field is not present
	}
	buf := bytes.NewBuffer(nil)
	_, err := io.Copy(buf, file)
	if err != nil {
		return ppf, 500, err // IO error
	}
	content := buf.Bytes()
	filename, err := fnFromContent(content)
	if err != nil {
		return ppf, 400, err // validation error (bad request)
	}
	var f string
	if repo == "" {
		f = filename
	} else {
		f = repo + "/" + filename
	}
	if !server.AllowOverwrite {
		_, err = server.StorageBackend.GetObject(f)
		if err == nil {
			return ppf, 409, fmt.Errorf("%s already exists", f) // conflict
		}
	}
	return &packageOrProvenanceFile{filename, content, field}, 200, nil
}
