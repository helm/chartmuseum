package multitenant

func (s *MultiTenantServer) setRoutes() {
	// Server Info
	s.Router.Groups.ReadAccess.GET(s.p("/"), s.defaultHandler)
	s.Router.Groups.SysInfo.GET(s.p("/system/health"), s.getHealthCheckHandler)

	// Helm Chart Repository
	s.Router.Groups.ReadAccess.GET(s.p("/:repo/index.yaml"), s.getIndexFileRequestHandler)
	s.Router.Groups.ReadAccess.GET(s.p("/:repo/charts/:filename"), s.getStorageObjectRequestHandler)
}
