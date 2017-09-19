package chartmuseum

import (
	"fmt"
	"strings"

	"github.com/chartmuseum/chartmuseum/pkg/repo"

	"github.com/gin-gonic/gin"
)

var (
	objectSavedResponse        = gin.H{"saved": true}
	objectDeletedResponse      = gin.H{"deleted": true}
	notFoundErrorResponse      = gin.H{"error": "not found"}
	badExtensionErrorResponse  = gin.H{"error": "unsupported file extension"}
	alreadyExistsErrorResponse = gin.H{"error": "file already exists"}
)

func (server *Server) getIndexFileRequestHandler(c *gin.Context) {
	err := server.syncRepositoryIndex()
	if err != nil {
		c.JSON(500, errorResponse(err))
		return
	}
	c.Data(200, repo.IndexFileContentType, server.RepositoryIndex.Raw)
}

func (server *Server) getAllChartsRequestHandler(c *gin.Context) {
	err := server.syncRepositoryIndex()
	if err != nil {
		c.JSON(500, errorResponse(err))
		return
	}
	c.JSON(200, server.RepositoryIndex.Entries)
}

func (server *Server) getChartRequestHandler(c *gin.Context) {
	name := c.Param("name")
	err := server.syncRepositoryIndex()
	if err != nil {
		c.JSON(500, errorResponse(err))
		return
	}
	chart := server.RepositoryIndex.Entries[name]
	if chart == nil {
		c.JSON(404, notFoundErrorResponse)
		return
	}
	c.JSON(200, chart)
}

func (server *Server) getChartVersionRequestHandler(c *gin.Context) {
	name := c.Param("name")
	version := c.Param("version")
	if version == "latest" {
		version = ""
	}
	err := server.syncRepositoryIndex()
	if err != nil {
		c.JSON(500, errorResponse(err))
		return
	}
	chartVersion, err := server.RepositoryIndex.Get(name, version)
	if err != nil {
		c.JSON(404, notFoundErrorResponse)
		return
	}
	c.JSON(200, chartVersion)
}

func (server *Server) deleteChartVersionRequestHandler(c *gin.Context) {
	name := c.Param("name")
	version := c.Param("version")
	filename := repo.ChartPackageFilenameFromNameVersion(name, version)
	server.Logger.Debugw("Deleting package from storage",
		"package", filename,
	)
	err := server.StorageBackend.DeleteObject(filename)
	if err != nil {
		c.JSON(404, notFoundErrorResponse)
		return
	}
	provFilename := repo.ProvenanceFilenameFromNameVersion(name, version)
	server.StorageBackend.DeleteObject(provFilename) // ignore error here, may be no prov file
	c.JSON(200, objectDeletedResponse)
}

func (server *Server) getStorageObjectRequestHandler(c *gin.Context) {
	filename := c.Param("filename")
	isChartPackage := strings.HasSuffix(filename, repo.ChartPackageFileExtension)
	isProvenanceFile := strings.HasSuffix(filename, repo.ProvenanceFileExtension)
	if !isChartPackage && !isProvenanceFile {
		c.JSON(500, badExtensionErrorResponse)
		return
	}
	object, err := server.StorageBackend.GetObject(filename)
	if err != nil {
		c.JSON(404, notFoundErrorResponse)
		return
	}
	if isProvenanceFile {
		c.Data(200, repo.ProvenanceFileContentType, object.Content)
		return
	}
	c.Data(200, repo.ChartPackageContentType, object.Content)
}

func (server *Server) postPackageRequestHandler(c *gin.Context) {
	content, err := c.GetRawData()
	if err != nil {
		c.JSON(500, errorResponse(err))
		return
	}
	filename, err := repo.ChartPackageFilenameFromContent(content)
	if err != nil {
		c.JSON(500, errorResponse(err))
		return
	}
	_, err = server.StorageBackend.GetObject(filename)
	if err == nil {
		c.JSON(500, alreadyExistsErrorResponse)
		return
	}
	server.Logger.Debugw("Adding package to storage",
		"package", filename,
	)
	err = server.StorageBackend.PutObject(filename, content)
	if err != nil {
		c.JSON(500, errorResponse(err))
		return
	}
	c.JSON(201, objectSavedResponse)
}

func (server *Server) postProvenanceFileRequestHandler(c *gin.Context) {
	content, err := c.GetRawData()
	if err != nil {
		c.JSON(500, errorResponse(err))
		return
	}
	filename, err := repo.ProvenanceFilenameFromContent(content)
	if err != nil {
		c.JSON(500, errorResponse(err))
		return
	}
	_, err = server.StorageBackend.GetObject(filename)
	if err == nil {
		c.JSON(500, alreadyExistsErrorResponse)
		return
	}
	server.Logger.Debugw("Adding provenance file to storage",
		"provenance_file", filename,
	)
	err = server.StorageBackend.PutObject(filename, content)
	if err != nil {
		c.JSON(500, errorResponse(err))
		return
	}
	c.JSON(201, objectSavedResponse)
}

func errorResponse(err error) map[string]interface{} {
	errResp := gin.H{"error": fmt.Sprintf("%s", err)}
	return errResp
}
