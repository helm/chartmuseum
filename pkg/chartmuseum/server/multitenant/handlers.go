package multitenant

import (
	"github.com/gin-gonic/gin"
)

var (
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

func (server *MultiTenantServer) getIndexFileRequestHandler(c *gin.Context) {
	repo := server.getContextParam(c, "repo")
	indexFile, err := server.getIndexFile(repo)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	c.Data(200, indexFileContentType, indexFile.Raw)
}

func (server *MultiTenantServer) getStorageObjectRequestHandler(c *gin.Context) {
	repo := server.getContextParam(c, "repo")
	filename := server.getContextParam(c, "filename")
	storageObject, err := server.getStorageObject(repo, filename)
	if err != nil {
		c.JSON(err.Status, gin.H{"error": err.Message})
		return
	}
	c.Data(200, storageObject.ContentType, storageObject.Content)
}
