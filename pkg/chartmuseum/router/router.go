package router

import (
	"fmt"
	"regexp"

	"github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"

	"github.com/atarantini/ginrequestid"
	"github.com/gin-contrib/location"
	"github.com/gin-gonic/gin"
	"github.com/zsais/go-gin-prometheus"
)

type (
	// Router handles all incoming HTTP requests
	Router struct {
		*gin.Engine
	}
)

// NewRouter creates a new Router instance
func NewRouter(logger *logger.Logger, enableMetrics bool) *Router {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(location.Default(), ginrequestid.RequestId(), loggingMiddleware(logger), gin.Recovery())
	if enableMetrics {
		p := ginprometheus.NewPrometheus("chartmuseum")
		p.ReqCntURLLabelMappingFn = mapURLWithParamsBackToRouteTemplate
		p.Use(engine)
	}
	return &Router{engine}
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
