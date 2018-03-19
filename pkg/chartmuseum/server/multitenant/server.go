package multitenant

import (
	"fmt"
	"math/rand"
	pathutil "path"
	"strings"
	"sync"
	"time"

	"github.com/kubernetes-helm/chartmuseum/pkg/cache"
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"

	"github.com/gin-gonic/gin"
)

var (
	PathPrefix string
)

type (
	// MultiTenantServer contains a Logger, Router, storage backend and object cache
	MultiTenantServer struct {
		Logger            *cm_logger.Logger
		Router            *cm_router.Router
		StorageBackend    storage.Backend
		Cache             cache.Store
		Depth             int
		IndexLimit        int
		IndexCache        map[string]*cachedIndexFile
		IndexCacheKeyLock *sync.Mutex
	}

	// MultiTenantServerOptions are options for constructing a MultiTenantServer
	MultiTenantServerOptions struct {
		Logger         *cm_logger.Logger
		Router         *cm_router.Router
		StorageBackend storage.Backend
		Cache          cache.Store
		Depth          int
	}
)

// NewMultiTenantServer creates a new MultiTenantServer instance
func NewMultiTenantServer(options MultiTenantServerOptions) (*MultiTenantServer, error) {
	server := &MultiTenantServer{
		Logger:            options.Logger,
		Router:            options.Router,
		StorageBackend:    options.StorageBackend,
		Cache:             options.Cache,
		Depth:             options.Depth,
		IndexCache:        map[string]*cachedIndexFile{},
		IndexCacheKeyLock: &sync.Mutex{},
	}

	server.setRoutes()

	return server, nil
}

// Listen TODO
func (server *MultiTenantServer) Listen(port int) {
	server.Router.Start(port)
}

// get URL param, "repo" gets special treatment since it is augmented
// in the headers by the router
func (server *MultiTenantServer) getContextParam(c *gin.Context, param string) string {
	if param == "repo" {
		return c.Request.Header.Get("ChartMuseum-Repo")
	}
	return c.Param(param)
}

// simple helper to prepend the necessary path prefix for each route
// based on server.Depth, ":arg1/:arg2" etc added for extended route matching
func (server *MultiTenantServer) p(path string) string {
	var a []string
	for i := 1; i <= server.Depth; i++ {
		a = append(a, fmt.Sprintf(":arg%d", i))
	}
	dynamicParamsPath := "/" + strings.Join(a, "/")
	path = strings.Replace(path, "/:repo", dynamicParamsPath, 1)
	return pathutil.Join(PathPrefix, path)
}

// make the PathPrefix pretty much unguessable,
// incoming requests with this prefix will not be logged
func setPathPrefix() {
	charset := "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 40)
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	PathPrefix = "/" + string(b)
}

func init() {
	setPathPrefix()
}
