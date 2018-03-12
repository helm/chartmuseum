package multitenant

func (server *MultiTenantServer) setRoutes() {
	server.Router.Groups.ReadAccess.GET("/", server.defaultHandler)
}
