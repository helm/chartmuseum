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

package multitenant

import (
	"fmt"
	"os"
	"sync"
	"time"

	"helm.sh/chartmuseum/pkg/cache"
	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_router "helm.sh/chartmuseum/pkg/chartmuseum/router"
	cm_repo "helm.sh/chartmuseum/pkg/repo"

	"github.com/chartmuseum/storage"
	cm_storage "github.com/chartmuseum/storage"
	"github.com/gin-gonic/gin"
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
		TimestampTolerance     time.Duration
		ExternalCacheStore     cache.Store
		InternalCacheStore     map[string]*cacheEntry
		MaxStorageObjects      int
		IndexLimit             int
		AllowOverwrite         bool
		AllowForceOverwrite    bool
		APIEnabled             bool
		DisableDelete          bool
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
		TimestampTolerance     time.Duration
		ChartURL               string
		ChartPostFormFieldName string
		ProvPostFormFieldName  string
		MaxStorageObjects      int
		IndexLimit             int
		GenIndex               bool
		AllowOverwrite         bool
		AllowForceOverwrite    bool
		EnableAPI              bool
		DisableDelete          bool
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
		TimestampTolerance:     options.TimestampTolerance,
		ExternalCacheStore:     options.ExternalCacheStore,
		InternalCacheStore:     map[string]*cacheEntry{},
		MaxStorageObjects:      options.MaxStorageObjects,
		IndexLimit:             options.IndexLimit,
		ChartURL:               chartURL,
		ChartPostFormFieldName: options.ChartPostFormFieldName,
		ProvPostFormFieldName:  options.ProvPostFormFieldName,
		AllowOverwrite:         options.AllowOverwrite,
		AllowForceOverwrite:    options.AllowForceOverwrite,
		APIEnabled:             options.EnableAPI,
		DisableDelete:          options.DisableDelete,
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
