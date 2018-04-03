package multitenant

import (
	"fmt"
	"os"
	"strings"
	"sync"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"
)

var (
	echo = fmt.Print
	exit = os.Exit
)

type (
	// MultiTenantServer contains a Logger, Router, storage backend and object cache
	MultiTenantServer struct {
		Logger                 *cm_logger.Logger
		Router                 *cm_router.Router
		StorageBackend         storage.Backend
		IndexLimit             int
		AllowOverwrite         bool
		APIEnabled             bool
		ChartURL               string
		ChartPostFormFieldName string
		ProvPostFormFieldName  string
		Limiter                chan struct{}
		IndexCache             map[string]*cachedIndexFile
		IndexCacheKeyLock      *sync.Mutex
	}

	// MultiTenantServerOptions are options for constructing a MultiTenantServer
	MultiTenantServerOptions struct {
		Logger                 *cm_logger.Logger
		Router                 *cm_router.Router
		StorageBackend         storage.Backend
		ChartURL               string
		ChartPostFormFieldName string
		ProvPostFormFieldName  string
		IndexLimit             int
		GenIndex               bool
		AllowOverwrite         bool
		EnableAPI              bool
	}
)

// NewMultiTenantServer creates a new MultiTenantServer instance
func NewMultiTenantServer(options MultiTenantServerOptions) (*MultiTenantServer, error) {
	var chartURL string
	if options.ChartURL != "" {
		chartURL = strings.TrimSuffix(options.ChartURL, "/") + options.Router.ContextPath
	}

	server := &MultiTenantServer{
		Logger:                 options.Logger,
		Router:                 options.Router,
		StorageBackend:         options.StorageBackend,
		IndexLimit:             options.IndexLimit,
		ChartURL:               chartURL,
		ChartPostFormFieldName: options.ChartPostFormFieldName,
		ProvPostFormFieldName:  options.ProvPostFormFieldName,
		AllowOverwrite:         options.AllowOverwrite,
		APIEnabled:             options.EnableAPI,
		Limiter:                make(chan struct{}, options.IndexLimit),
		IndexCache:             map[string]*cachedIndexFile{},
		IndexCacheKeyLock:      &sync.Mutex{},
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
	echo(string(server.IndexCache[""].RepositoryIndex.Raw[:]))
	exit(0)
}
