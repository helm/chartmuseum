package chartmuseum

import (
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
	mt "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/server/multitenant"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"
)

type (
	// ServerOptions are options for constructing a Server
	ServerOptions struct {
		StorageBackend         storage.Backend
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
		AllowOverwrite         bool
		EnableMetrics          bool
		AnonymousGet           bool
		GenIndex               bool
		IndexLimit             int
		Depth                  int
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

	router := cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Username:      options.Username,
		Password:      options.Password,
		ContextPath:   options.ContextPath,
		TlsCert:       options.TlsCert,
		TlsKey:        options.TlsKey,
		EnableMetrics: options.EnableMetrics,
		AnonymousGet:  options.AnonymousGet,
		Depth:         options.Depth,
	})

	server, err := mt.NewMultiTenantServer(mt.MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         options.StorageBackend,
		ChartURL:               options.ChartURL,
		ChartPostFormFieldName: options.ChartPostFormFieldName,
		ProvPostFormFieldName:  options.ProvPostFormFieldName,
		IndexLimit:             options.IndexLimit,
		GenIndex:               options.GenIndex,
		EnableAPI:              options.EnableAPI,
		AllowOverwrite:         options.AllowOverwrite,
	})

	return server, err
}
