package chartmuseum

import (
	"context"
	"fmt"
	"regexp"
	"sync"

	"github.com/kubernetes-helm/chartmuseum/pkg/repo"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"

	"github.com/atarantini/ginrequestid"
	"github.com/gin-gonic/gin"
	"github.com/zsais/go-gin-prometheus"
	helm_repo "k8s.io/helm/pkg/repo"
)

type (
	// Router handles all incoming HTTP requests
	Router struct {
		*gin.Engine
	}

	// Server contains a Logger, Router, storage backend and object cache
	Server struct {
		Logger                  *Logger
		Router                  *Router
		RepositoryIndex         *repo.Index
		StorageBackend          storage.Backend
		StorageCache            []storage.Object
		AllowOverwrite          bool
		TlsCert                 string
		TlsKey                  string
		ChartPostFormFieldName  string
		ProvPostFormFieldName   string
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
		ChartURL               string
		TlsCert                string
		TlsKey                 string
		Username               string
		Password               string
		ChartPostFormFieldName string
		ProvPostFormFieldName  string
	}

	fetchedObjects struct {
		objects []storage.Object
		err     error
	}
	indexRegeneration struct {
		index *repo.Index
		err   error
	}
)

func mapURLWithParamsBackToRouteTemplate(c *gin.Context) string {
	url := c.Request.URL.String()
	for _, p := range c.Params {
		re := regexp.MustCompile(fmt.Sprintf(`(^.*?)/\b%s\b(.*$)`, regexp.QuoteMeta(p.Value)))
		url = re.ReplaceAllString(url, fmt.Sprintf(`$1/:%s$2`, p.Key))
	}
	return url
}

// NewRouter creates a new Router instance
func NewRouter(logger *Logger, username string, password string, enableMetrics bool) *Router {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(ginrequestid.RequestId(), loggingMiddleware(logger), gin.Recovery())
	if username != "" && password != "" {
		users := make(map[string]string)
		users[username] = password
		engine.Use(gin.BasicAuthForRealm(users, "ChartMuseum"))
	}
	if enableMetrics {
		p := ginprometheus.NewPrometheus("chartmuseum")
		// For every route containing parameters (e.g. `/charts/:filename`, `/api/charts/:name/:version`, etc)
		// the actual parameter values will be replaced by their name, to minimize the cardinality of the
		// `chartmuseum_requests_total{url=..}` Prometheus counter.
		p.ReqCntURLLabelMappingFn = mapURLWithParamsBackToRouteTemplate
		p.Use(engine)
	}
	return &Router{engine}
}

// NewServer creates a new Server instance
func NewServer(options ServerOptions) (*Server, error) {
	logger, err := NewLogger(options.LogJSON, options.Debug)
	if err != nil {
		return new(Server), nil
	}

	router := NewRouter(logger, options.Username, options.Password, options.EnableMetrics)

	server := &Server{
		Logger:                 logger,
		Router:                 router,
		RepositoryIndex:        repo.NewIndex(options.ChartURL),
		StorageBackend:         options.StorageBackend,
		StorageCache:           []storage.Object{},
		AllowOverwrite:         options.AllowOverwrite,
		TlsCert:                options.TlsCert,
		TlsKey:                 options.TlsKey,
		ChartPostFormFieldName: options.ChartPostFormFieldName,
		ProvPostFormFieldName:  options.ProvPostFormFieldName,
		regenerationLock:       &sync.Mutex{},
		fetchedObjectsLock:     &sync.Mutex{},
	}

	server.setRoutes(options.EnableAPI)

	// prime the cache
	_, err = server.syncRepositoryIndex(&gin.Context{})
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

// getChartList fetches from the server and accumulates concurrent requests to be fulfilled all at once.
func (server *Server) getChartList(c *gin.Context) <-chan fetchedObjects {
	ch := make(chan fetchedObjects, 1)

	server.fetchedObjectsLock.Lock()
	server.fetchedObjectsChans = append(server.fetchedObjectsChans, ch)

	if len(server.fetchedObjectsChans) == 1 {
		// this unlock is wanted, while fetching the list, allow other channeled requests to be added
		server.fetchedObjectsLock.Unlock()

		objects, err := server.fetchChartsInStorage(c)

		server.fetchedObjectsLock.Lock()
		// flush every other consumer that also wanted the index
		for _, foCh := range server.fetchedObjectsChans {
			foCh <- fetchedObjects{objects, err}
		}
		server.fetchedObjectsChans = nil
	}
	server.fetchedObjectsLock.Unlock()

	return ch
}

func (server *Server) regenerateRepositoryIndex(c *gin.Context, diff storage.ObjectSliceDiff, storageObjects []storage.Object) <-chan indexRegeneration {
	ch := make(chan indexRegeneration, 1)

	server.regenerationLock.Lock()
	server.regeneratedIndexesChans = append(server.regeneratedIndexesChans, ch)

	if len(server.regeneratedIndexesChans) == 1 {
		server.regenerationLock.Unlock()

		index, err := server.regenerateRepositoryIndexWorker(c, diff, storageObjects)

		server.regenerationLock.Lock()
		for _, riCh := range server.regeneratedIndexesChans {
			riCh <- indexRegeneration{index, err}
		}
		server.regeneratedIndexesChans = nil
	}
	server.regenerationLock.Unlock()

	return ch
}

/*
syncRepositoryIndex is the workhorse of maintaining a coherent index cache. It is optimized for multiple requests
comming in a short period. When two requests for the backing store arrive, only the first is served, and other consumers receive the
result of this request. This allows very fast updates in constant time. See getChartList() and regenerateRepositoryIndex().
*/
func (server *Server) syncRepositoryIndex(c *gin.Context) (*repo.Index, error) {
	fo := <-server.getChartList(c)

	if fo.err != nil {
		return nil, fo.err
	}

	diff := storage.GetObjectSliceDiff(server.StorageCache, fo.objects)

	// return fast if no changes
	if !diff.Change {
		return server.RepositoryIndex, nil
	}

	ir := <-server.regenerateRepositoryIndex(c, diff, fo.objects)

	return ir.index, ir.err
}

func (server *Server) fetchChartsInStorage(c *gin.Context) ([]storage.Object, error) {
	server.Logger.Debugc(c, "Fetching chart list from storage")
	allObjects, err := server.StorageBackend.ListObjects()
	if err != nil {
		return []storage.Object{}, err
	}

	// filter out storage objects that dont have extension used for chart packages (.tgz)
	filteredObjects := []storage.Object{}
	for _, object := range allObjects {
		if object.HasExtension(repo.ChartPackageFileExtension) {
			filteredObjects = append(filteredObjects, object)
		}
	}

	return filteredObjects, nil
}

func (server *Server) regenerateRepositoryIndexWorker(c *gin.Context, diff storage.ObjectSliceDiff, storageObjects []storage.Object) (*repo.Index, error) {
	server.Logger.Debugc(c, "Regenerating index.yaml")
	index := &repo.Index{
		IndexFile: server.RepositoryIndex.IndexFile,
		Raw:       server.RepositoryIndex.Raw,
		ChartURL:  server.RepositoryIndex.ChartURL,
	}

	for _, object := range diff.Removed {
		err := server.removeIndexObject(c, index, object)
		if err != nil {
			return nil, err
		}
	}

	for _, object := range diff.Updated {
		err := server.updateIndexObject(c, index, object)
		if err != nil {
			return nil, err
		}
	}

	// Parallelize retrieval of added objects to improve speed
	err := server.addIndexObjectsAsync(c, index, diff.Added)
	if err != nil {
		return nil, err
	}

	err = index.Regenerate()
	if err != nil {
		return nil, err
	}

	// It is very important that these two stay in sync as they reflect the same reality. StorageCache serves
	// as object modification time cache, and RepositoryIndex is the canonical cached index.
	server.RepositoryIndex = index
	server.StorageCache = storageObjects

	server.Logger.Debugc(c, "index.yaml regenerated")
	return index, nil
}

func (server *Server) removeIndexObject(c *gin.Context, index *repo.Index, object storage.Object) error {
	chartVersion, err := server.getObjectChartVersion(object, false)
	if err != nil {
		return server.checkInvalidChartPackageError(c, object, err, "removed")
	}
	server.Logger.Debugc(c, "Removing chart from index",
		"name", chartVersion.Name,
		"version", chartVersion.Version,
	)
	index.RemoveEntry(chartVersion)
	return nil
}

func (server *Server) updateIndexObject(c *gin.Context, index *repo.Index, object storage.Object) error {
	chartVersion, err := server.getObjectChartVersion(object, true)
	if err != nil {
		return server.checkInvalidChartPackageError(c, object, err, "updated")
	}
	server.Logger.Debugc(c, "Updating chart in index",
		"name", chartVersion.Name,
		"version", chartVersion.Version,
	)
	index.UpdateEntry(chartVersion)
	return nil
}

func (server *Server) addIndexObjectsAsync(c *gin.Context, index *repo.Index, objects []storage.Object) error {
	numObjects := len(objects)
	if numObjects == 0 {
		return nil
	}

	server.Logger.Debugc(c, "Loading charts packages from storage (this could take awhile)",
		"total", numObjects,
	)

	type cvResult struct {
		cv  *helm_repo.ChartVersion
		err error
	}

	cvChan := make(chan cvResult)

	// Provide a mechanism to short-circuit object downloads in case of error
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, object := range objects {
		go func(o storage.Object) {
			select {
			case <-ctx.Done():
				return
			default:
				chartVersion, err := server.getObjectChartVersion(o, true)
				if err != nil {
					err = server.checkInvalidChartPackageError(c, o, err, "added")
				}
				if err != nil {
					cancel()
				}
				cvChan <- cvResult{chartVersion, err}
			}
		}(object)
	}

	for validCount := 0; validCount < numObjects; validCount++ {
		cvRes := <-cvChan
		if cvRes.err != nil {
			return cvRes.err
		}
		if cvRes.cv == nil {
			continue
		}
		server.Logger.Debugc(c, "Adding chart to index",
			"name", cvRes.cv.Name,
			"version", cvRes.cv.Version,
		)
		index.AddEntry(cvRes.cv)
	}

	return nil
}

func (server *Server) getObjectChartVersion(object storage.Object, load bool) (*helm_repo.ChartVersion, error) {
	if load {
		var err error
		object, err = server.StorageBackend.GetObject(object.Path)
		if err != nil {
			return nil, err
		}
		if len(object.Content) == 0 {
			return nil, repo.ErrorInvalidChartPackage
		}
	}
	return repo.ChartVersionFromStorageObject(object)
}

func (server *Server) checkInvalidChartPackageError(c *gin.Context, object storage.Object, err error, action string) error {
	if err == repo.ErrorInvalidChartPackage {
		server.Logger.Warnc(c, "Invalid package in storage",
			"action", action,
			"package", object.Path,
		)
		return nil
	}
	return err
}
