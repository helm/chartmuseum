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

func (server *Server) setMultiTenancyRoutes() {
	server.Router.GET("/", func(c *gin.Context) {
		msg := "This ChartMuseum install is running in multi-repo mode."
		c.Data(200, "text/plain", []byte(msg))
	})
}
