package router

import (
	"fmt"
	"regexp"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"

	"github.com/gin-gonic/gin"
	"github.com/zsais/go-gin-prometheus"
)

type (
	// Router handles all incoming HTTP requests
	Router struct {
		*gin.Engine
		Logger      *cm_logger.Logger
		Routes      []*Route
		TlsCert     string
		TlsKey      string
		ContextPath string
		Depth       int
	}

	// RouterOptions TODO
	RouterOptions struct {
		Logger        *cm_logger.Logger
		Username      string
		Password      string
		ContextPath   string
		TlsCert       string
		TlsKey        string
		PathPrefix    string
		EnableMetrics bool
		AnonymousGet  bool
		Depth         int
	}

	// Route TODO
	Route struct {
		Method  string
		Path    string
		Handler gin.HandlerFunc
	}
)

// NewRouter creates a new Router instance
func NewRouter(options RouterOptions) *Router {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(requestWrapper(options.Logger))

	if options.EnableMetrics {
		p := ginprometheus.NewPrometheus("chartmuseum")
		p.ReqCntURLLabelMappingFn = mapURLWithParamsBackToRouteTemplate
		p.Use(engine)
	}

	router := &Router{
		Engine:      engine,
		Routes:      []*Route{},
		Logger:      options.Logger,
		TlsCert:     options.TlsCert,
		TlsKey:      options.TlsKey,
		ContextPath: options.ContextPath,
		Depth:       options.Depth,
	}

	router.NoRoute(router.matchRoute)

	return router
}

func (router *Router) Start(port int) {
	router.Logger.Infow("Starting ChartMuseum",
		"port", port,
	)
	if router.TlsCert != "" && router.TlsKey != "" {
		router.Logger.Fatal(router.RunTLS(fmt.Sprintf(":%d", port), router.TlsCert, router.TlsKey))
	} else {
		router.Logger.Fatal(router.Run(fmt.Sprintf(":%d", port)))
	}
}

// SetRoutes applies list of routes
func (router *Router) SetRoutes(routes []*Route) {
	router.Routes = routes
}

func (router *Router) globalHandler(c *gin.Context) {
	c.JSON(200, gin.H{"msg": "test"})
}

/*
mapURLWithParamsBackToRouteTemplate is a valid ginprometheus ReqCntURLLabelMappingFn.
For every route containing parameters (e.g. `/charts/:filename`, `/api/charts/:name/:version`, etc)
the actual parameter values will be replaced by their name, to minimize the cardinality of the
`chartmuseum_requests_total{url=..}` Prometheus counter.
*/
func mapURLWithParamsBackToRouteTemplate(c *gin.Context) string {
	url := c.Request.URL.String()
	for _, p := range c.Params {
		re := regexp.MustCompile(fmt.Sprintf(`(^.*?)/\b%s\b(.*$)`, regexp.QuoteMeta(p.Value)))
		url = re.ReplaceAllString(url, fmt.Sprintf(`$1/:%s$2`, p.Key))
	}
	return url
}
