/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package chartmuseum

import (
	"strings"
	"time"

	"github.com/chartmuseum/storage"
	"helm.sh/chartmuseum/pkg/cache"
	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_router "helm.sh/chartmuseum/pkg/chartmuseum/router"
	mt "helm.sh/chartmuseum/pkg/chartmuseum/server/multitenant"
)

type (
	// ServerOptions are options for constructing a Server
	ServerOptions struct {
		StorageBackend         storage.Backend
		ExternalCacheStore     cache.Store
		TimestampTolerance     time.Duration
		ChartURL               string
		TlsCert                string
		TlsKey                 string
		TlsCACert              string
		Username               string
		Password               string
		ChartPostFormFieldName string
		ProvPostFormFieldName  string
		ContextPath            string
		LogJSON                bool
		LogHealth              bool
		LogLatencyInteger      bool
		Debug                  bool
		EnableAPI              bool
		UseStatefiles          bool
		AllowOverwrite         bool
		DisableDelete          bool
		AllowForceOverwrite    bool
		EnableMetrics          bool
		AnonymousGet           bool
		GenIndex               bool
		MaxStorageObjects      int
		IndexLimit             int
		Depth                  int
		MaxUploadSize          int
		BearerAuth             bool
		AuthRealm              string
		AuthService            string
		AuthCertPath           string
		AuthActionsSearchPath  string
		DepthDynamic           bool
		CORSAllowOrigin        string
		ReadTimeout            int
		WriteTimeout           int
		// EnforceSemver2 represents if the museum server always accept the Chart with [valid semantic version 2](https://semver.org/)
		// More refers to : https://github.com/helm/chartmuseum/issues/320
		EnforceSemver2 bool
		CacheInterval  time.Duration
		Host           string
		Version        string
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
		Logger:            		logger,
		LogLatencyInteger: 		options.LogLatencyInteger,
		Username:          		options.Username,
		Password:          		options.Password,
		ContextPath:       		contextPath,
		TlsCert:           		options.TlsCert,
		TlsKey:            		options.TlsKey,
		TlsCACert:         		options.TlsCACert,
		LogHealth:         		options.LogHealth,
		EnableMetrics:     		options.EnableMetrics,
		AnonymousGet:      		options.AnonymousGet,
		Depth:             		options.Depth,
		MaxUploadSize:     		options.MaxUploadSize,
		BearerAuth:        		options.BearerAuth,
		AuthRealm:         		options.AuthRealm,
		AuthService:       		options.AuthService,
		AuthCertPath:      		options.AuthCertPath,
		AuthActionsSearchPath: 	options.AuthActionsSearchPath,
		DepthDynamic:      		options.DepthDynamic,
		CORSAllowOrigin:   		options.CORSAllowOrigin,
		ReadTimeout:       		options.ReadTimeout,
		WriteTimeout:      		options.WriteTimeout,
		Host:              		options.Host,
	})

	server, err := mt.NewMultiTenantServer(mt.MultiTenantServerOptions{
		Logger:                 logger,
		Router:                 router,
		StorageBackend:         options.StorageBackend,
		ExternalCacheStore:     options.ExternalCacheStore,
		TimestampTolerance:     options.TimestampTolerance,
		ChartURL:               strings.TrimSuffix(options.ChartURL, "/"),
		ChartPostFormFieldName: options.ChartPostFormFieldName,
		ProvPostFormFieldName:  options.ProvPostFormFieldName,
		MaxStorageObjects:      options.MaxStorageObjects,
		IndexLimit:             options.IndexLimit,
		GenIndex:               options.GenIndex,
		EnableAPI:              options.EnableAPI,
		DisableDelete:          options.DisableDelete,
		UseStatefiles:          options.UseStatefiles,
		AllowOverwrite:         options.AllowOverwrite,
		AllowForceOverwrite:    options.AllowForceOverwrite,
		EnforceSemver2:         options.EnforceSemver2,
		Version:                options.Version,
		CacheInterval:          options.CacheInterval,
	})

	return server, err
}
