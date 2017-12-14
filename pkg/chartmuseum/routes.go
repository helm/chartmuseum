package chartmuseum

import "github.com/gin-gonic/gin"

func (server *Server) setRoutes(username string, password string, enableAPI bool) {
	// Routes that never use basic HTTP Auth can be applied directly to the default Router
	server.Router.GET("/", server.getHealthCheck)

	// Routes that can use basic HTTP Auth must be applied to the basicAuthGroup Router Group
	basicAuthGroup := server.Router.Group("")
	if username != "" && password != "" {
		users := make(map[string]string)
		users[username] = password
		basicAuthGroup.Use(gin.BasicAuthForRealm(users, "ChartMuseum"))
	}

	// Helm Chart Repository
	basicAuthGroup.GET("/index.yaml", server.getIndexFileRequestHandler)
	basicAuthGroup.GET("/charts/:filename", server.getStorageObjectRequestHandler)

	// Chart Manipulation
	if enableAPI {
		basicAuthGroup.GET("/api/charts", server.getAllChartsRequestHandler)
		basicAuthGroup.POST("/api/charts", server.postRequestHandler)
		basicAuthGroup.POST("/api/prov", server.postProvenanceFileRequestHandler)
		basicAuthGroup.GET("/api/charts/:name", server.getChartRequestHandler)
		basicAuthGroup.GET("/api/charts/:name/:version", server.getChartVersionRequestHandler)
		basicAuthGroup.DELETE("/api/charts/:name/:version", server.deleteChartVersionRequestHandler)
	}
}
