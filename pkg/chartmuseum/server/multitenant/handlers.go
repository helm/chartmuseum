package multitenant

import (
	"fmt"
	"strings"

	"github.com/kubernetes-helm/chartmuseum/pkg/repo"

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
<p>This feature is still and progress, and is not considered stable.</p>
<p>Please run without the --multitenant flag to disable this.</p>
</body>
</html>
	`)
)

func (server *MultiTenantServer) defaultHandler(c *gin.Context) {
	c.Data(200, "text/html", warningHTML)
}

func (server *MultiTenantServer) getIndexFileRequestHandler(c *gin.Context) {
	orgName := c.Param("org")
	repoName := c.Param("repo")
	prefix := fmt.Sprintf("%s/%s", orgName, repoName)

	objects, err := server.StorageBackend.ListObjects(prefix)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("%s", err)})
		return
	}

	index := repo.NewIndex("")
	for _, object := range objects {
		op := object.Path
		objectPath := fmt.Sprintf("%s/%s", prefix, op)
		object, err = server.StorageBackend.GetObject(objectPath)
		if err != nil {
			// TODO handle err
			continue
		}
		chartVersion, err := repo.ChartVersionFromStorageObject(object)
		if err != nil {
			// TODO handle err
			continue
		}
		chartVersion.URLs[0] = fmt.Sprintf("charts/%s", op)
		index.AddEntry(chartVersion)
	}

	index.Regenerate()
	c.Data(200, repo.IndexFileContentType, index.Raw)
}

func (server *MultiTenantServer) getStorageObjectRequestHandler(c *gin.Context) {
	orgName := c.Param("org")
	repoName := c.Param("repo")
	prefix := fmt.Sprintf("%s/%s", orgName, repoName)

	filename := c.Param("filename")
	isChartPackage := strings.HasSuffix(filename, repo.ChartPackageFileExtension)
	isProvenanceFile := strings.HasSuffix(filename, repo.ProvenanceFileExtension)
	if !isChartPackage && !isProvenanceFile {
		c.JSON(500, gin.H{"error": "unsupported file extension"})
		return
	}
	objectPath := fmt.Sprintf("%s/%s", prefix, filename)
	object, err := server.StorageBackend.GetObject(objectPath)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	if isProvenanceFile {
		c.Data(200, repo.ProvenanceFileContentType, object.Content)
		return
	}
	c.Data(200, repo.ChartPackageContentType, object.Content)
}
