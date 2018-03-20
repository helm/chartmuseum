package singletenant

import (
	"fmt"
	"os"
	"sync"

	"github.com/kubernetes-helm/chartmuseum/pkg/cache"
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
	"github.com/kubernetes-helm/chartmuseum/pkg/repo"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"

	"github.com/gin-gonic/gin"
)

var (
	echo = fmt.Print
	exit = os.Exit
)

type (
	// SingleTenantServer contains a Logger, Router, storage backend and object cache
	SingleTenantServer struct {
		Logger                  *cm_logger.Logger
		Router                  *cm_router.Router
		RepositoryIndex         *repo.Index
		StorageBackend          storage.Backend
		Cache                   cache.Store
		StorageCache            []storage.Object
		AllowOverwrite          bool
		APIEnabled              bool
		ChartPostFormFieldName  string
		ProvPostFormFieldName   string
		IndexLimit              int
		regenerationLock        *sync.Mutex
		fetchedObjectsLock      *sync.Mutex
		fetchedObjectsChans     []chan fetchedObjects
		regeneratedIndexesChans []chan indexRegeneration
	}

	// SingleTenantServerOptions are options for constructing a SingleTenantServer
	SingleTenantServerOptions struct {
		Logger                 *cm_logger.Logger
		Router                 *cm_router.Router
		StorageBackend         storage.Backend
		Cache                  cache.Store
		EnableAPI              bool
		AllowOverwrite         bool
		GenIndex               bool
		ChartURL               string
		ChartPostFormFieldName string
		ProvPostFormFieldName  string
		IndexLimit             int
	}
)

// NewSingleTenantServer creates a new SingleTenantServer instance
func NewSingleTenantServer(options SingleTenantServerOptions) (*SingleTenantServer, error) {
	server := &SingleTenantServer{
		Logger:                 options.Logger,
		Router:                 options.Router,
		RepositoryIndex:        repo.NewIndex(options.ChartURL),
		StorageBackend:         options.StorageBackend,
		Cache:                  options.Cache,
		StorageCache:           []storage.Object{},
		APIEnabled:             options.EnableAPI,
		AllowOverwrite:         options.AllowOverwrite,
		ChartPostFormFieldName: options.ChartPostFormFieldName,
		ProvPostFormFieldName:  options.ProvPostFormFieldName,
		IndexLimit:             options.IndexLimit,
		regenerationLock:       &sync.Mutex{},
		fetchedObjectsLock:     &sync.Mutex{},
	}

	server.setRoutes()

	// prime the cache
	log := server.Logger.ContextLoggingFn(&gin.Context{})
	_, err := server.syncRepositoryIndex(log)

	if options.GenIndex {
		server.genIndex()
	}

	return server, err
}

// Listen TODO
func (server *SingleTenantServer) Listen(port int) {
	server.Router.Start(port)
}

func (server *SingleTenantServer) genIndex() {
	echo(string(server.RepositoryIndex.Raw[:]))
	exit(0)
}

// simple helper to modify route definitions
func (server *SingleTenantServer) p(path string) string {
	return cm_router.PrefixRouteDefinition(path, 0)
}
