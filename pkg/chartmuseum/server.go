package chartmuseum

import (
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"
	"github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	"github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
	"github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/server/singletenant"
)

type (
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
		GenIndex               bool
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

	Server interface {
		Listen(port int)
	}
)

// NewServer creates a new Server instance
func NewServer(options ServerOptions) (Server, error) {
	loggerOptions := logger.LoggerOptions{
		Debug:   options.Debug,
		LogJSON: options.LogJSON,
	}
	logger, err := logger.NewLogger(loggerOptions)
	if err != nil {
		return nil, nil
	}

	routerOptions := router.RouterOptions{
		Logger:        logger,
		Username:      options.Username,
		Password:      options.Password,
		ContextPath:   options.ContextPath,
		TlsCert:       options.TlsCert,
		TlsKey:        options.TlsKey,
		EnableAPI:     options.EnableAPI,
		EnableMetrics: options.EnableMetrics,
	}
	router := router.NewRouter(routerOptions)

	if options.EnableMultiTenancy {
		panic("please run without the --multitenant flag")
	} else {
		singleTenantServerOptions := singletenant.SingleTenantServerOptions{
			Logger:                 logger,
			Router:                 router,
			StorageBackend:         options.StorageBackend,
			EnableAPI:              options.EnableAPI,
			AllowOverwrite:         options.AllowOverwrite,
			GenIndex:               options.GenIndex,
			ChartURL:               options.ChartURL,
			ChartPostFormFieldName: options.ChartPostFormFieldName,
			ProvPostFormFieldName:  options.ProvPostFormFieldName,
			ContextPath:            options.ContextPath,
			IndexLimit:             options.IndexLimit,
		}
		singleTenantServer, err := singletenant.NewSingleTenantServer(singleTenantServerOptions)
		return singleTenantServer, err
	}
}
