package chartmuseum

func (server *Server) setRoutes() {
	// Helm Chart Repository
	server.Router.GET("/index.yaml", server.getIndexFileRequestHandler)
	server.Router.GET("/charts/:filename", server.getStorageObjectRequestHandler)

	// Chart Manipulation
	server.Router.GET("/api/charts", server.getAllChartsRequestHandler)
	server.Router.POST("/api/charts", server.postPackageRequestHandler)
	server.Router.POST("/api/prov", server.postProvenanceFileRequestHandler)
	server.Router.GET("/api/charts/:name", server.getChartRequestHandler)
	server.Router.GET("/api/charts/:name/:version", server.getChartVersionRequestHandler)
	server.Router.DELETE("/api/charts/:name/:version", server.deleteChartVersionRequestHandler)
}
