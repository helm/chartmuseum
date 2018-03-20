package multitenant

import (
	"sync"

	"github.com/kubernetes-helm/chartmuseum/pkg/cache"
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"
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

// simple helper to modify route definitions
func (server *MultiTenantServer) p(path string) string {
	return cm_router.PrefixRouteDefinition(path, server.Depth)
}
