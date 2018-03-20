package multitenant

func (s *MultiTenantServer) setRoutes() {
	// Server Info
	s.Router.Groups.ReadAccess.GET(s.p("/"), s.defaultHandler)
	s.Router.Groups.SysInfo.GET(s.p("/system/health"), s.getHealthCheckHandler)

	// Helm Chart Repository
	s.Router.Groups.ReadAccess.GET(s.p("/:repo/index.yaml"), s.getIndexFileRequestHandler)
	s.Router.Groups.ReadAccess.GET(s.p("/:repo/charts/:filename"), s.getStorageObjectRequestHandler)

	// Chart Manipulation
	if s.APIEnabled {
		s.Router.Groups.ReadAccess.GET(s.p("/api/:repo/charts"), s.getAllChartsRequestHandler)
		s.Router.Groups.ReadAccess.GET(s.p("/api/:repo/charts/:name"), s.getChartRequestHandler)
		s.Router.Groups.ReadAccess.GET(s.p("/api/:repo/charts/:name/:version"), s.getChartVersionRequestHandler)
		// TODO: enable these
		//s.Router.Groups.WriteAccess.POST(s.p("/api/charts"), s.postRequestHandler)
		//s.Router.Groups.WriteAccess.POST(s.p("/api/prov"), s.postProvenanceFileRequestHandler)
		//s.Router.Groups.WriteAccess.DELETE(s.p("/api/charts/:name/:version"), s.deleteChartVersionRequestHandler)
	}
}
