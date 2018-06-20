package chartmuseum

import (
	"strings"

	"github.com/kubernetes-helm/chartmuseum/pkg/cache"
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
	mt "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/server/multitenant"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"
)

type (
	// ServerOptions are options for constructing a Server
	ServerOptions struct {
		StorageBackend         storage.Backend
		ExternalCacheStore     cache.Store
		ChartURL               string
		TlsCert                string
		TlsKey                 string
		Username               string
		Password               string
		ChartPostFormFieldName string
		ProvPostFormFieldName  string
		ContextPath            string
		LogJSON                bool
		Debug                  bool
		EnableAPI              bool
		UseStatefiles          bool
		AllowOverwrite         bool
		EnableMetrics          bool
		AnonymousGet           bool
		GenIndex               bool
		MaxStorageObjects      int
		IndexLimit             int
		Depth                  int
		MaxUploadSize          int
	}

	// Server is a generic interface for web servers
	Server interface {
		Listen(port int)
	}
)

// NewServer creates a new Server instance
func NewServer(options ServerOptions) (Server, error) {
	logger, err := cm_logger.NewLogger(cm_logger.LoggerOptions{
		Debug:   options.Debug,
		LogJSON: options.LogJSON,
	})
	if err != nil {
		return nil, err
	}

	contextPath := strings.TrimSuffix(options.ContextPath, "/")
	if contextPath != "" && !strings.HasPrefix(contextPath, "/") {
		contextPath = "/" + contextPath
	}

	router := cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Username:      options.Username,
		Password:      options.Password,
		ContextPath:   contextPath,
		TlsCert:       options.TlsCert,
		TlsKey:        options.TlsKey,
		EnableMetrics: options.EnableMetrics,
		AnonymousGet:  options.AnonymousGet,
		Depth:         options.Depth,
		MaxUploadSize: options.MaxUploadSize,
	})

	server, err := mt.NewMultiTenantServer(mt.MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         options.StorageBackend,
		ExternalCacheStore:     options.ExternalCacheStore,
		ChartURL:               strings.TrimSuffix(options.ChartURL, "/"),
		ChartPostFormFieldName: options.ChartPostFormFieldName,
		ProvPostFormFieldName:  options.ProvPostFormFieldName,
		MaxStorageObjects:      options.MaxStorageObjects,
		IndexLimit:             options.IndexLimit,
		GenIndex:               options.GenIndex,
		EnableAPI:              options.EnableAPI,
		UseStatefiles:          options.UseStatefiles,
		AllowOverwrite:         options.AllowOverwrite,
	})

	return server, err
}
