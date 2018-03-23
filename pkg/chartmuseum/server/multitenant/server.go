package multitenant

import (
	"sync"

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
		IndexLimit        int
		Limiter           chan struct{}
		IndexCache        map[string]*cachedIndexFile
		IndexCacheKeyLock *sync.Mutex
	}

	// MultiTenantServerOptions are options for constructing a MultiTenantServer
	MultiTenantServerOptions struct {
		Logger         *cm_logger.Logger
		Router         *cm_router.Router
		StorageBackend storage.Backend
		IndexLimit     int
	}
)

// NewMultiTenantServer creates a new MultiTenantServer instance
func NewMultiTenantServer(options MultiTenantServerOptions) (*MultiTenantServer, error) {
	server := &MultiTenantServer{
		Logger:            options.Logger,
		Router:            options.Router,
		StorageBackend:    options.StorageBackend,
		IndexCache:        map[string]*cachedIndexFile{},
		IndexCacheKeyLock: &sync.Mutex{},
	}

	if options.IndexLimit != 0 {
		server.Limiter = make(chan struct{}, options.IndexLimit)
	}

	server.Router.SetRoutes(server.Routes())

	return server, nil
}

// Listen TODO
func (server *MultiTenantServer) Listen(port int) {
	server.Router.Start(port)
}
