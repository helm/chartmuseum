package multitenant

import (
	"fmt"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/kubernetes-helm/chartmuseum/pkg/cache"
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
	cm_repo "github.com/kubernetes-helm/chartmuseum/pkg/repo"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"
	cm_storage "github.com/kubernetes-helm/chartmuseum/pkg/storage"
)

var (
	echo = fmt.Print
	exit = os.Exit
)

const (
	defaultFormField = "chart"
	defaultProvField = "prov"
)

type (
	// MultiTenantServer contains a Logger, Router, storage backend and object cache
	MultiTenantServer struct {
		Logger                 *cm_logger.Logger
		Router                 *cm_router.Router
		StorageBackend         storage.Backend
		ExternalCacheStore     cache.Store
		InternalCacheStore     map[string]*cacheEntry
		MaxStorageObjects      int
		IndexLimit             int
		AllowOverwrite         bool
		APIEnabled             bool
		UseStatefiles          bool
		ChartURL               string
		ChartPostFormFieldName string
		ProvPostFormFieldName  string
		Limiter                chan struct{}
		Tenants                map[string]*tenantInternals
		TenantCacheKeyLock     *sync.Mutex
	}

	// MultiTenantServerOptions are options for constructing a MultiTenantServer
	MultiTenantServerOptions struct {
		Logger                 *cm_logger.Logger
		Router                 *cm_router.Router
		StorageBackend         storage.Backend
		ExternalCacheStore     cache.Store
		ChartURL               string
		ChartPostFormFieldName string
		ProvPostFormFieldName  string
		MaxStorageObjects      int
		IndexLimit             int
		GenIndex               bool
		AllowOverwrite         bool
		EnableAPI              bool
		UseStatefiles          bool
	}

	tenantInternals struct {
		FetchedObjectsLock      *sync.Mutex
		RegenerationLock        *sync.Mutex
		FetchedObjectsChans     []chan fetchedObjects
		RegeneratedIndexesChans []chan indexRegeneration
	}

	fetchedObjects struct {
		objects []cm_storage.Object
		err     error
	}

	indexRegeneration struct {
		index *cm_repo.Index
		err   error
	}
)

// NewMultiTenantServer creates a new MultiTenantServer instance
func NewMultiTenantServer(options MultiTenantServerOptions) (*MultiTenantServer, error) {
	var chartURL string
	if options.ChartURL != "" {
		chartURL = options.ChartURL + options.Router.ContextPath
	}

	server := &MultiTenantServer{
		Logger:                 options.Logger,
		Router:                 options.Router,
		StorageBackend:         options.StorageBackend,
		ExternalCacheStore:     options.ExternalCacheStore,
		InternalCacheStore:     map[string]*cacheEntry{},
		MaxStorageObjects:      options.MaxStorageObjects,
		IndexLimit:             options.IndexLimit,
		ChartURL:               chartURL,
		ChartPostFormFieldName: options.ChartPostFormFieldName,
		ProvPostFormFieldName:  options.ProvPostFormFieldName,
		AllowOverwrite:         options.AllowOverwrite,
		APIEnabled:             options.EnableAPI,
		UseStatefiles:          options.UseStatefiles,
		Limiter:                make(chan struct{}, options.IndexLimit),
		Tenants:                map[string]*tenantInternals{},
		TenantCacheKeyLock:     &sync.Mutex{},
	}

	server.Router.SetRoutes(server.Routes())
	err := server.primeCache()

	if options.GenIndex && server.Router.Depth == 0 {
		server.genIndex()
	}

	return server, err
}

// Listen starts the router on a given port
func (server *MultiTenantServer) Listen(port int) {
	server.Router.Start(port)
}

func (server *MultiTenantServer) genIndex() {
	log := server.Logger.ContextLoggingFn(&gin.Context{})
	entry, err := server.initCacheEntry(log, "")
	if err != nil {
		panic(err)
	}
	echo(string(entry.RepoIndex.Raw[:]))
	exit(0)
}
