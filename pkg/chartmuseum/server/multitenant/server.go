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
	"strings"
	"sync"
	"time"

	cm_storage "github.com/chartmuseum/storage"
	"github.com/gin-gonic/gin"

	"helm.sh/chartmuseum/pkg/cache"
	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_router "helm.sh/chartmuseum/pkg/chartmuseum/router"
	cm_repo "helm.sh/chartmuseum/pkg/repo"
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
		StorageBackend         cm_storage.Backend
		TimestampTolerance     time.Duration
		ExternalCacheStore     cache.Store
		InternalCacheStore     memoryCacheStore
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
		Version                string
		Limiter                chan struct{}
		Tenants                map[string]*tenantInternals
		TenantCacheKeyLock     *sync.Mutex
		CacheInterval          time.Duration
		EventChan              chan event
		ChartLimits            *ObjectsPerChartLimit
		ArtifactHubRepoID      map[string]string
		// Deprecated: see https://github.com/helm/chartmuseum/issues/485 for more info
		EnforceSemver2  bool
		WebTemplatePath string
	}

	ObjectsPerChartLimit struct {
		*sync.Mutex
		Limit int
	}

	// MultiTenantServerOptions are options for constructing a MultiTenantServer
	MultiTenantServerOptions struct {
		Logger                 *cm_logger.Logger
		Router                 *cm_router.Router
		StorageBackend         cm_storage.Backend
		ExternalCacheStore     cache.Store
		TimestampTolerance     time.Duration
		ChartURL               string
		ChartPostFormFieldName string
		ProvPostFormFieldName  string
		Version                string
		MaxStorageObjects      int
		IndexLimit             int
		GenIndex               bool
		AllowOverwrite         bool
		AllowForceOverwrite    bool
		EnableAPI              bool
		DisableDelete          bool
		UseStatefiles          bool
		CacheInterval          time.Duration
		PerChartLimit          int
		ArtifactHubRepoID      map[string]string
		WebTemplatePath        string
		// Deprecated: see https://github.com/helm/chartmuseum/issues/485 for more info
		EnforceSemver2 bool
	}

	tenantInternals struct {
		FetchedObjectsLock      *sync.Mutex
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
	var l *ObjectsPerChartLimit
	if options.PerChartLimit > 0 {
		l = &ObjectsPerChartLimit{
			Mutex: &sync.Mutex{},
			Limit: options.PerChartLimit,
		}
	}

	server := &MultiTenantServer{
		Logger:                 options.Logger,
		Router:                 options.Router,
		StorageBackend:         options.StorageBackend,
		TimestampTolerance:     options.TimestampTolerance,
		ExternalCacheStore:     options.ExternalCacheStore,
		InternalCacheStore:     memoryCacheStore{},
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
		EnforceSemver2:         options.EnforceSemver2,
		Version:                options.Version,
		Limiter:                make(chan struct{}, options.IndexLimit),
		Tenants:                map[string]*tenantInternals{},
		TenantCacheKeyLock:     &sync.Mutex{},
		CacheInterval:          options.CacheInterval,
		ChartLimits:            l,
		WebTemplatePath:        options.WebTemplatePath,
		ArtifactHubRepoID:      options.ArtifactHubRepoID,
	}

	if server.WebTemplatePath != "" {
		// check if template file exists to avoid panic when calling LoadHTMLGlob
		templateFilesExist := server.CheckTemplateFilesExist(server.WebTemplatePath, server.Logger)
		if templateFilesExist {
			server.Router.LoadHTMLGlob(fmt.Sprintf("%s/*.html", server.WebTemplatePath))
		} else {
			server.Logger.Warnf("No template files found in %s", server.WebTemplatePath)
		}
	}

	server.Router.SetRoutes(server.Routes())
	err := server.primeCache()

	if options.GenIndex && server.Router.Depth == 0 {
		server.genIndex()
	}

	server.EventChan = make(chan event, server.IndexLimit)
	go server.startEventListener()
	server.initCacheTimer()

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

func (server *MultiTenantServer) CheckTemplateFilesExist(path string, logger *cm_logger.Logger) bool {
	// check if template file exists
	webTemplateFolder, err := os.Open(path)
	if err != nil {
		logger.Errorf("Failed to open template folder %s", path)
		return false
	}
	templates, err := webTemplateFolder.Readdir(0)
	if err != nil {
		server.Logger.Errorf("Error reading template files from %s", path)
		return false
	}
	for _, template := range templates {
		if strings.HasSuffix(template.Name(), ".html") {
			return true
		}
	}
	return false
}
