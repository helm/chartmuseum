package multitenant

func (server *MultiTenantServer) setRoutes() {
	// Server Info
	server.Router.Groups.ReadAccess.GET(p("/"), server.defaultHandler)

	// Helm Chart Repository
	server.Router.Groups.ReadAccess.GET(p("/:org/:repo/index.yaml"), server.getIndexFileRequestHandler)
	server.Router.Groups.ReadAccess.GET(p("/:org/:repo/charts/:filename"), server.getStorageObjectRequestHandler)
}
