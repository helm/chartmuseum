package multitenant

func (server *MultiTenantServer) setRoutes() {
	// Server Info
	server.Router.Groups.ReadAccess.GET("/", server.defaultHandler)

	// Helm Chart Repository
	server.Router.Groups.ReadAccess.GET("/r/:org/:repo/index.yaml", server.getIndexFileRequestHandler)
	server.Router.Groups.ReadAccess.GET("/r/:org/:repo/charts/:filename", server.getStorageObjectRequestHandler)
}
