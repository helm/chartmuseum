package singletenant

func (server *SingleTenantServer) setRoutes() {
	// Server Info
	server.Router.Groups.ReadAccess.GET("/", server.getWelcomePageHandler)
	server.Router.Groups.SysInfo.GET("/health", server.getHealthCheckHandler)

	// Helm Chart Repository
	server.Router.Groups.ReadAccess.GET("/index.yaml", server.getIndexFileRequestHandler)
	server.Router.Groups.ReadAccess.GET("/charts/:filename", server.getStorageObjectRequestHandler)

	// Chart Manipulation
	if server.APIEnabled {
		server.Router.Groups.ReadAccess.GET("/api/charts", server.getAllChartsRequestHandler)
		server.Router.Groups.ReadAccess.GET("/api/charts/:name", server.getChartRequestHandler)
		server.Router.Groups.ReadAccess.GET("/api/charts/:name/:version", server.getChartVersionRequestHandler)
		server.Router.Groups.WriteAccess.POST("/api/charts", server.postRequestHandler)
		server.Router.Groups.WriteAccess.POST("/api/prov", server.postProvenanceFileRequestHandler)
		server.Router.Groups.WriteAccess.DELETE("/api/charts/:name/:version", server.deleteChartVersionRequestHandler)
	}
}
