package router

import (
	"fmt"
	"regexp"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"

	"github.com/atarantini/ginrequestid"
	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
	"github.com/zsais/go-gin-prometheus"
)

type (
	// Router handles all incoming HTTP requests
	Router struct {
		*gin.Engine
		Groups  *RouterGroups
		Logger  *cm_logger.Logger
		TlsCert string
		TlsKey  string
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
	}
)

// NewRouter creates a new Router instance
func NewRouter(options RouterOptions) *Router {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	// Middleware
	engine.Use(location.Default(), ginrequestid.RequestId(), loggingMiddleware(options.Logger, options.PathPrefix),
		gin.Recovery(), prefixPathMiddleware(engine, options.PathPrefix))

	if options.EnableMetrics {
		p := ginprometheus.NewPrometheus("chartmuseum")
		p.ReqCntURLLabelMappingFn = mapURLWithParamsBackToRouteTemplate
		p.Use(engine)
	}
	router := &Router{
		Engine:  engine,
		Groups:  new(RouterGroups),
		Logger:  options.Logger,
		TlsCert: options.TlsCert,
		TlsKey:  options.TlsKey,
	}
	routerGroupsOptions := RouterGroupsOptions{
		Logger:       options.Logger,
		Router:       router,
		Username:     options.Username,
		Password:     options.Password,
		ContextPath:  options.ContextPath,
		AnonymousGet: options.AnonymousGet,
	}
	router.Groups = NewRouterGroups(routerGroupsOptions)
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
