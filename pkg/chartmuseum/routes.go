package chartmuseum

import (
	"github.com/gin-gonic/gin"
)

func (server *Server) setRoutes(username string, password string, enableAPI bool) {
	sysInfoGroup := &server.Router.RouterGroup
	readAccessGroup := &server.Router.RouterGroup
	writeAccessGroup := &server.Router.RouterGroup

	// Reconfigure read-access, write-access groups if basic auth is enabled
	if username != "" && password != "" {
		basicAuthGroup := server.Router.Group("")
		users := make(map[string]string)
		users[username] = password
		basicAuthGroup.Use(gin.BasicAuthForRealm(users, "ChartMuseum"))
		writeAccessGroup = basicAuthGroup
		if server.AnonymousGet {
			server.Logger.Debug("Anonymous GET enabled")
		} else {
			readAccessGroup = basicAuthGroup
		}
	}

	// Multi-Tenancy POC
	orgAccessGroup := server.Router.Group("/mt")
	orgAccessGroup.Use(databaseMiddleware(server.Database))
	{
		// Fetch index.yaml, .tgz, .prov files from storage
		orgAccessGroup.GET("/:org/:repo/index.yaml", server.getOrgRepoIndexFileRequestHandler)
		orgAccessGroup.GET("/:org/:repo/charts/:filename", server.getOrgRepoStorageObjectRequestHandler)

		if enableAPI {
			// Org operations
			orgAccessGroup.GET("/", server.getOrgsRequestHandler)
			orgAccessGroup.POST("/", server.createOrgRequestHandler)
			orgAccessGroup.GET("/:org", server.getOrgRequestHandler)
			orgAccessGroup.DELETE("/:org", server.deleteOrgRequestHandler)

			// Org-owned Repo operations
			orgAccessGroup.POST("/:org", server.createRepoRequestHandler)
			orgAccessGroup.GET("/:org/:repo", server.getRepoRequestHandler)
			orgAccessGroup.DELETE("/:org/:repo", server.deleteRepoRequestHandler)

			// TODO: ChartMuseum CRUD API per Org-Repo combination
			//orgAccessGroup.GET("/:org/:repo/api/charts", server.getAllChartsRequestHandler)
			//orgAccessGroup.GET("/:org/:repo/api/charts/:name", server.getChartRequestHandler)
			//orgAccessGroup.GET("/:org/:repo/api/charts/:name/:version", server.getChartVersionRequestHandler)
			//orgAccessGroup.POST("/:org/:repo/api/charts", server.postRequestHandler)
			//orgAccessGroup.POST("/:org/:repo/api/prov", server.postProvenanceFileRequestHandler)
			//orgAccessGroup.DELETE("/:org/:repo/api/charts/:name/:version", server.deleteChartVersionRequestHandler)
		}
	}

	// Server Info
	sysInfoGroup.GET("/", server.getWelcomePageHandler)
	sysInfoGroup.GET("/health", server.getHealthCheckHandler)

	// Helm Chart Repository
	readAccessGroup.GET("/index.yaml", server.getIndexFileRequestHandler)
	readAccessGroup.GET("/charts/:filename", server.getStorageObjectRequestHandler)

	// Chart Manipulation
	if enableAPI {
		readAccessGroup.GET("/api/charts", server.getAllChartsRequestHandler)
		readAccessGroup.GET("/api/charts/:name", server.getChartRequestHandler)
		readAccessGroup.GET("/api/charts/:name/:version", server.getChartVersionRequestHandler)
		writeAccessGroup.POST("/api/charts", server.postRequestHandler)
		writeAccessGroup.POST("/api/prov", server.postProvenanceFileRequestHandler)
		writeAccessGroup.DELETE("/api/charts/:name/:version", server.deleteChartVersionRequestHandler)
	}
}
