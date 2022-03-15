/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package router

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"

	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"

	cm_auth "github.com/chartmuseum/auth"
	limits "github.com/gin-contrib/size"
	"github.com/gin-gonic/gin"
	ginprometheus "github.com/zsais/go-gin-prometheus"
)

type (
	// Router handles all incoming HTTP requests
	Router struct {
		*gin.Engine
		Logger          *cm_logger.Logger
		Authorizer      *cm_auth.Authorizer
		Routes          []*Route
		TlsCert         string
		TlsKey          string
		TlsCACert       string
		ContextPath     string
		Depth           int
		DepthDynamic    bool
		CORSAllowOrigin string
		ReadTimeout     time.Duration
		WriteTimeout    time.Duration
		Host            string
		WebTemplatePath string
	}

	// RouterOptions are options for constructing a Router
	RouterOptions struct {
		Logger                *cm_logger.Logger
		LogLatencyInteger     bool
		Username              string
		Password              string
		ContextPath           string
		TlsCert               string
		TlsKey                string
		TlsCACert             string
		PathPrefix            string
		LogHealth             bool
		EnableMetrics         bool
		AnonymousGet          bool
		Depth                 int
		MaxUploadSize         int
		BearerAuth            bool
		AuthRealm             string
		AuthService           string
		AuthCertPath          string
		AuthActionsSearchPath string
		DepthDynamic          bool
		ReadTimeout           int
		WriteTimeout          int
		CORSAllowOrigin       string
		Host                  string
	}

	// Route represents an application route
	Route struct {
		Method  string
		Path    string
		Handler gin.HandlerFunc
		Action  string
	}
)

// NewRouter creates a new Router instance
func NewRouter(options RouterOptions) *Router {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.RedirectTrailingSlash = false // This was causing /health to 301 to /health/
	engine.Use(gin.Recovery())
	engine.Use(requestWrapper(options.Logger, options.LogHealth, options.LogLatencyInteger))
	engine.Use(limits.RequestSizeLimiter(int64(options.MaxUploadSize)))

	if options.EnableMetrics {
		p := ginprometheus.NewPrometheus("chartmuseum")
		p.ReqCntURLLabelMappingFn = mapURLWithParamsBackToRouteTemplate
		p.Use(engine)
	}

	router := &Router{
		Engine:          engine,
		Routes:          []*Route{},
		Logger:          options.Logger,
		TlsCert:         options.TlsCert,
		TlsKey:          options.TlsKey,
		TlsCACert:       options.TlsCACert,
		ContextPath:     options.ContextPath,
		Depth:           options.Depth,
		DepthDynamic:    options.DepthDynamic,
		CORSAllowOrigin: options.CORSAllowOrigin,
		ReadTimeout:     time.Duration(options.ReadTimeout) * time.Second,
		WriteTimeout:    time.Duration(options.WriteTimeout) * time.Second,
		Host:            options.Host,
	}

	var err error
	var authorizer *cm_auth.Authorizer

	// if BearerAuth is true, looks for required inputs.
	// example input:
	// --bearer-auth
	// --auth-realm="https://my.site.io/oauth2/token"
	// --auth-service="my.site.io"
	// --auth-cert-path="./certs/authorization-server-cert.pem"
	if options.BearerAuth {
		if options.AuthRealm == "" {
			router.Logger.Fatal("Missing Auth Realm")
		}
		if options.AuthService == "" {
			router.Logger.Fatal("Missing Auth Service")
		}
		if options.AuthCertPath == "" {
			router.Logger.Fatal("Missing Auth Server Public Cert Path")
		}

		authorizer, err = cm_auth.NewAuthorizer(&cm_auth.AuthorizerOptions{
			Realm:                    options.AuthRealm,
			Service:                  options.AuthService,
			PublicKeyPath:            options.AuthCertPath,
			AllowedActionsSearchPath: options.AuthActionsSearchPath,
		})
	} else if options.Username != "" && options.Password != "" {
		authorizer, err = cm_auth.NewAuthorizer(&cm_auth.AuthorizerOptions{
			Realm:    "ChartMuseum",
			Username: options.Username,
			Password: options.Password,
		})
	}

	if err != nil {
		router.Logger.Fatal(err)
	}

	if authorizer != nil && options.AnonymousGet {
		authorizer.AnonymousActions = []string{cm_auth.PullAction}
	}

	router.Authorizer = authorizer

	router.NoRoute(router.rootHandler)

	return router
}

func (router *Router) Start(port int) {
	router.Logger.Infow("Starting ChartMuseum",
		"host", router.Host, "port", port,
	)

	server := http.Server{
		Addr:         fmt.Sprintf("%s:%d", router.Host, port),
		Handler:      router,
		ReadTimeout:  router.ReadTimeout,
		WriteTimeout: router.WriteTimeout,
	}

	if router.TlsCert != "" && router.TlsKey != "" {
		if router.TlsCACert != "" {
			keypair, _ := tls.LoadX509KeyPair(router.TlsCert, router.TlsKey)
			certpool := x509.NewCertPool()
			capem, _ := ioutil.ReadFile(router.TlsCACert)
			if !certpool.AppendCertsFromPEM(capem) {
				router.Logger.Fatal("Can't parse CA certificate file")
			}
			server.TLSConfig = &tls.Config{
				Certificates: []tls.Certificate{keypair},
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    certpool,
			}
			router.Logger.Fatal(server.ListenAndServeTLS("", ""))
		} else {
			router.Logger.Fatal(server.ListenAndServeTLS(router.TlsCert, router.TlsKey))
		}
	} else {
		router.Logger.Fatal(server.ListenAndServe())
	}
}

// SetRoutes applies list of routes
func (router *Router) SetRoutes(routes []*Route) {
	router.Routes = routes
}

// all incoming requests are passed through this handler
func (router *Router) rootHandler(c *gin.Context) {
	route, params := match(router.Routes, c.Request.Method, c.Request.URL.Path, router.ContextPath, router.Depth,
		router.DepthDynamic)
	if route == nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	c.Params = params

	if route.Action != "" && router.Authorizer != nil {
		authHeader := c.Request.Header.Get("Authorization")

		namespace := c.Param("repo")
		if namespace == "" {
			namespace = cm_auth.DefaultNamespace
		}

		permissions, err := router.Authorizer.Authorize(authHeader, route.Action, namespace)
		if err != nil {
			router.Logger.Error(err)
			c.JSON(500, gin.H{"error": "internal server error"})
			return
		}

		if !permissions.Allowed {
			if permissions.WWWAuthenticateHeader != "" {
				c.Header("WWW-Authenticate", permissions.WWWAuthenticateHeader)
			}
			c.JSON(401, gin.H{"error": "unauthorized"})
			return
		}
	}

	if checkApiRoute(c.Request.URL.Path) && router.CORSAllowOrigin != "" {
		c.Header("Access-Control-Allow-Origin", router.CORSAllowOrigin)
	}

	route.Handler(c)
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
