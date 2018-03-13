package multitenant

import (
	"math/rand"
	pathutil "path"
	"time"

	"github.com/kubernetes-helm/chartmuseum/pkg/cache"
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"
)

var (
	PathPrefix string
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

// simple helper to prepend the necessary path prefix for each route
func p(path string) string {
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
