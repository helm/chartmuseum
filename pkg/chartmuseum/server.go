package chartmuseum

import (
	"fmt"
	"sync"

	"github.com/kubernetes-helm/chartmuseum/pkg/repo"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"
	"github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	"github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"

	"github.com/gin-gonic/gin"
)

type (
	// Server contains a Logger, Router, storage backend and object cache
	Server struct {
		Logger                  *logger.Logger
		Router                  *router.Router
		RepositoryIndex         *repo.Index
		StorageBackend          storage.Backend
		StorageCache            []storage.Object
		AllowOverwrite          bool
		MultiTenancyEnabled     bool
		AnonymousGet            bool
		TlsCert                 string
		TlsKey                  string
		ChartPostFormFieldName  string
		ProvPostFormFieldName   string
		IndexLimit              int
		regenerationLock        *sync.Mutex
		fetchedObjectsLock      *sync.Mutex
		fetchedObjectsChans     []chan fetchedObjects
		regeneratedIndexesChans []chan indexRegeneration
	}

	// ServerOptions are options for constructing a Server
	ServerOptions struct {
		StorageBackend         storage.Backend
		LogJSON                bool
		Debug                  bool
		EnableAPI              bool
		AllowOverwrite         bool
		EnableMetrics          bool
		EnableMultiTenancy     bool
		AnonymousGet           bool
		ChartURL               string
		TlsCert                string
		TlsKey                 string
		Username               string
		Password               string
		ChartPostFormFieldName string
		ProvPostFormFieldName  string
		IndexLimit             int
		ContextPath            string
	}
)

// NewServer creates a new Server instance
func NewServer(options ServerOptions) (*Server, error) {
	logger, err := logger.NewLogger(options.LogJSON, options.Debug)
	if err != nil {
		return new(Server), nil
	}

	router := router.NewRouter(logger, options.EnableMetrics)

	server := &Server{
		Logger:                 logger,
		Router:                 router,
		RepositoryIndex:        repo.NewIndex(options.ChartURL),
		StorageBackend:         options.StorageBackend,
		StorageCache:           []storage.Object{},
		AllowOverwrite:         options.AllowOverwrite,
		MultiTenancyEnabled:    options.EnableMultiTenancy,
		AnonymousGet:           options.AnonymousGet,
		TlsCert:                options.TlsCert,
		TlsKey:                 options.TlsKey,
		ChartPostFormFieldName: options.ChartPostFormFieldName,
		ProvPostFormFieldName:  options.ProvPostFormFieldName,
		IndexLimit:             options.IndexLimit,
		regenerationLock:       &sync.Mutex{},
		fetchedObjectsLock:     &sync.Mutex{},
	}

	server.setRoutes(options.Username, options.Password, options.EnableAPI, options.ContextPath)

	// prime the cache
	log := logger.ContextLoggingFn(&gin.Context{})
	_, err = server.syncRepositoryIndex(log)
	return server, err
}

// Listen starts server on a given port
func (server *Server) Listen(port int) {
	server.Logger.Infow("Starting ChartMuseum",
		"port", port,
	)
	if server.TlsCert != "" && server.TlsKey != "" {
		server.Logger.Fatal(server.Router.RunTLS(fmt.Sprintf(":%d", port), server.TlsCert, server.TlsKey))
	} else {
		server.Logger.Fatal(server.Router.Run(fmt.Sprintf(":%d", port)))
	}
}
