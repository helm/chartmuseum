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
	rootRoutePrefix   string
	systemRoutePrefix string
	apiRoutePrefix    string
	repoRoutePrefix   string
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
		Depth         int
	}
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

// PrefixRouteDefinition prepends the necessary path prefix for each
// route based on depth, ":arg1/:arg2" etc added for extended route matching
func PrefixRouteDefinition(path string, depth int) string {
	var prefix string

	// TODO: remove check of /health once singletenant goes away
	if path == "/" || path == "/health" {
		prefix = pathutil.Join(rootRoutePrefix, path)

	} else if strings.HasPrefix(path, "/system/") {
		prefix = pathutil.Join(systemRoutePrefix, path)

	} else if strings.Contains(path, "/:repo/") {
		hasRepoPrefix := strings.HasPrefix(path, "/:repo/")

		var a []string
		for i := 1; i <= depth; i++ {
			a = append(a, fmt.Sprintf(":arg%d", i))
		}
		dynamicParamsPath := "/" + strings.Join(a, "/")
		path = strings.Replace(path, "/:repo", dynamicParamsPath, 1)

		if hasRepoPrefix {
			prefix = pathutil.Join(repoRoutePrefix, path)
		}
	}

	if strings.HasPrefix(path, "/api/") {
		prefix = pathutil.Join(apiRoutePrefix, path)
	}

	return prefix
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
	systemRoutePrefix = getRandPathPrefix()
	apiRoutePrefix = getRandPathPrefix()
	repoRoutePrefix = getRandPathPrefix()
}
