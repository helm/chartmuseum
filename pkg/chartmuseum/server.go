package chartmuseum

import (
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_router "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/router"
	st "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/server/singletenant"
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

	router := cm_router.NewRouter(cm_router.RouterOptions{
		Logger:        logger,
		Username:      options.Username,
		Password:      options.Password,
		ContextPath:   options.ContextPath,
		TlsCert:       options.TlsCert,
		TlsKey:        options.TlsKey,
		EnableMetrics: options.EnableMetrics,
		AnonymousGet:  options.AnonymousGet,
	})

	if options.EnableMultiTenancy {
		panic("please run without the --multitenant flag")
	} else {
		singleTenantServer, err := st.NewSingleTenantServer(st.SingleTenantServerOptions{
			Logger:                 logger,
			Router:                 router,
			StorageBackend:         options.StorageBackend,
			EnableAPI:              options.EnableAPI,
			AllowOverwrite:         options.AllowOverwrite,
			GenIndex:               options.GenIndex,
			ChartURL:               options.ChartURL,
			ChartPostFormFieldName: options.ChartPostFormFieldName,
			ProvPostFormFieldName:  options.ProvPostFormFieldName,
			IndexLimit:             options.IndexLimit,
		})
		return singleTenantServer, err
	}
}
