package multitenant

import (
	"github.com/gin-gonic/gin"
)

var (
	healthCheckResponse = gin.H{"healthy": true}
	warningHTML = []byte(`<!DOCTYPE html>
<html>
<head>
<title>WARNING</title>
</head>
<body>
<h1>WARNING</h1>
<p>This ChartMuseum install is running in multitenancy mode.</p>
<p>This feature is still a work in progress, and is not considered stable.</p>
<p>Please run without the --multitenant flag to disable this.</p>
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

func (server *MultiTenantServer) defaultHandler(c *gin.Context) {
	c.Data(200, "text/html", warningHTML)
}

func (server *MultiTenantServer) getHealthCheckHandler(c *gin.Context) {
	c.JSON(200, healthCheckResponse)
}

func (server *MultiTenantServer) getIndexFileRequestHandler(c *gin.Context) {
	repo := c.GetString("repo")
	log := server.Logger.ContextLoggingFn(c)
	indexFile, err := server.getIndexFile(log, repo)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	c.Data(200, indexFileContentType, indexFile.Raw)
}

func (server *MultiTenantServer) getStorageObjectRequestHandler(c *gin.Context) {
	repo := c.GetString("repo")
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
	repo := c.GetString("repo")
	log := server.Logger.ContextLoggingFn(c)
	indexFile, err := server.getIndexFile(log, repo)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	c.JSON(200, indexFile.Entries)
}

func (server *MultiTenantServer) getChartRequestHandler(c *gin.Context) {
	repo := c.GetString("repo")
	name := c.Param("name")
	log := server.Logger.ContextLoggingFn(c)
	indexFile, err := server.getIndexFile(log, repo)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	chart := indexFile.Entries[name]
	if chart == nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.JSON(200, chart)
}

func (server *MultiTenantServer) getChartVersionRequestHandler(c *gin.Context) {
	repo := c.GetString("repo")
	name := c.Param("name")
	version := c.Param("version")
	if version == "latest" {
		version = ""
	}
	log := server.Logger.ContextLoggingFn(c)
	indexFile, err := server.getIndexFile(log, repo)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	chartVersion, getErr := indexFile.Get(name, version)
	if getErr != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.JSON(200, chartVersion)
}
