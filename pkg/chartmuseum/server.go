package chartmuseum

import (
	"github.com/kubernetes-helm/chartmuseum/pkg/cache"
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
	mt "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/server/multitenant"
	st "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/server/singletenant"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"
)

type (
	// ServerOptions are options for constructing a Server
	ServerOptions struct {
		StorageBackend         storage.Backend
		Cache                  cache.Store
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
		EnableMultiTenancy     bool
		AnonymousGet           bool
		GenIndex               bool
		IndexLimit             int
	}

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

	routerOptions := cm_router.RouterOptions{
		Logger:        logger,
		Username:      options.Username,
		Password:      options.Password,
		ContextPath:   options.ContextPath,
		TlsCert:       options.TlsCert,
		TlsKey:        options.TlsKey,
		EnableMetrics: options.EnableMetrics,
		AnonymousGet:  options.AnonymousGet,
	}
	if options.EnableMultiTenancy {
		routerOptions.PathPrefix = mt.PathPrefix
	}

	router := cm_router.NewRouter(routerOptions)

	var server Server

	if options.EnableMultiTenancy {
		logger.Debug("Multitenancy enabled")
		server, err = mt.NewMultiTenantServer(mt.MultiTenantServerOptions{
			Logger:         logger,
			Router:         router,
			StorageBackend: options.StorageBackend,
			Cache:          options.Cache,
		})
	} else {
		server, err = st.NewSingleTenantServer(st.SingleTenantServerOptions{
			Logger:                 logger,
			Router:                 router,
			StorageBackend:         options.StorageBackend,
			Cache:                  options.Cache,
			EnableAPI:              options.EnableAPI,
			AllowOverwrite:         options.AllowOverwrite,
			GenIndex:               options.GenIndex,
			ChartURL:               options.ChartURL,
			ChartPostFormFieldName: options.ChartPostFormFieldName,
			ProvPostFormFieldName:  options.ProvPostFormFieldName,
			IndexLimit:             options.IndexLimit,
		})
	}

	return server, err
}
