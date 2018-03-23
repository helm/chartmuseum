package router

import (
	"fmt"
	"math/rand"
	pathutil "path"
	"regexp"
	"strings"
	"time"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"

	"github.com/gin-gonic/gin"
	"github.com/zsais/go-gin-prometheus"
)

var (
	rootRoutePrefix string
	apiRoutePrefix  string
	repoRoutePrefix string
)

type (
	// Router handles all incoming HTTP requests
	Router struct {
		*gin.Engine
		Groups      *RouterGroups
		Logger      *cm_logger.Logger
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
		AccessGroup accessGroup
		Method      string
		Path        string
		Handler     gin.HandlerFunc
	}

	accessGroup string
)

const (
	ReadAccessGroup  accessGroup = "READ"
	WriteAccessGroup accessGroup = "WRITE"
	SystemInfoGroup  accessGroup = "SYSTEM"
)

// NewRouter creates a new Router instance
func NewRouter(options RouterOptions) *Router {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	// Add custom request wrapping middleware
	engine.Use(requestWrapper(options.Logger, engine, options.ContextPath, options.Depth))

	if options.EnableMetrics {
		p := ginprometheus.NewPrometheus("chartmuseum")
		p.ReqCntURLLabelMappingFn = mapURLWithParamsBackToRouteTemplate
		p.Use(engine)
	}
	router := &Router{
		Engine:      engine,
		Groups:      new(RouterGroups),
		Logger:      options.Logger,
		TlsCert:     options.TlsCert,
		TlsKey:      options.TlsKey,
		ContextPath: options.ContextPath,
		Depth:       options.Depth,
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

// SetRoutes applies list of routes, prepending the necessary path prefix for each
// route based on depth, ":arg1/:arg2" etc added for extended route matching
func (router *Router) SetRoutes(routes []Route) {
	for _, route := range routes {
		var accessGroup *gin.RouterGroup
		switch route.AccessGroup {
		case ReadAccessGroup:
			accessGroup = router.Groups.ReadAccess
		case WriteAccessGroup:
			accessGroup = router.Groups.WriteAccess
		case SystemInfoGroup:
			accessGroup = router.Groups.SysInfo
		}
		path := router.transformRoutePath(route.Path)
		accessGroup.Handle(route.Method, path, route.Handler)
	}
}

func (router *Router) transformRoutePath(path string) string {
	if path == "/" || path == "/health" {
		path = pathutil.Join(rootRoutePrefix, router.ContextPath, path)
	} else if strings.Contains(path, "/:repo/") {
		var a []string
		for i := 1; i <= router.Depth; i++ {
			a = append(a, fmt.Sprintf(":arg%d", i))
		}
		dynamicParamsPath := "/" + strings.Join(a, "/")
		path = strings.Replace(path, "/:repo", dynamicParamsPath, 1)
		if strings.HasPrefix(path, "/api/") {
			path = pathutil.Join(apiRoutePrefix, router.ContextPath, path)
		} else {
			path = pathutil.Join(repoRoutePrefix, router.ContextPath, path)
		}
	}
	return path
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

func getRandPathPrefix() string {
	charset := "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 40)
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return "/" + string(b)
}

// make the prefixes pretty much unguessable,
// incoming requests with these prefixes will not be treated properly
func init() {
	rootRoutePrefix = getRandPathPrefix()
	apiRoutePrefix = getRandPathPrefix()
	repoRoutePrefix = getRandPathPrefix()
}
