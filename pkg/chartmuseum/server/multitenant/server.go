package multitenant

import (
	"github.com/kubernetes-helm/chartmuseum/pkg/cache"
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"
)

type (
	// MultiTenantServer contains a Logger, Router, storage backend and object cache
	MultiTenantServer struct {
		Logger         *cm_logger.Logger
		Router         *cm_router.Router
		StorageBackend storage.Backend
		Cache          cache.Store
	}

	// MultiTenantServerOptions are options for constructing a MultiTenantServer
	MultiTenantServerOptions struct {
		Logger         *cm_logger.Logger
		Router         *cm_router.Router
		StorageBackend storage.Backend
		Cache          cache.Store
	}
)

// NewMultiTenantServer creates a new MultiTenantServer instance
func NewMultiTenantServer(options MultiTenantServerOptions) (*MultiTenantServer, error) {
	server := &MultiTenantServer{
		Logger:         options.Logger,
		Router:         options.Router,
		StorageBackend: options.StorageBackend,
		Cache:          options.Cache,
	}

	server.setRoutes()

	return server, nil
}

// Listen TODO
func (server *MultiTenantServer) Listen(port int) {
	server.Router.Start(port)
}
