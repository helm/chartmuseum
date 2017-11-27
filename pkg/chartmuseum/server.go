package chartmuseum

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kubernetes-helm/chartmuseum/pkg/repo"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"

	"github.com/atarantini/ginrequestid"
	"github.com/gin-gonic/gin"
	"github.com/zsais/go-gin-prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	helm_repo "k8s.io/helm/pkg/repo"
)

type (
	// Logger handles all logging from application
	Logger struct {
		*zap.SugaredLogger
	}

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

// NewLogger creates a new Logger instance
func NewLogger(json bool, debug bool) (*Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.DisableStacktrace = true
	config.Development = false
	if json {
		config.Encoding = "json"
	} else {
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	if !debug {
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	logger, err := config.Build()
	if err != nil {
		return new(Logger), err
	}
	defer logger.Sync()
	return &Logger{logger.Sugar()}, nil
}

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
	_, err = server.syncRepositoryIndex("bootstrap")
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

func loggingMiddleware(logger *Logger) gin.HandlerFunc {
	var requestCount int64
	return func(c *gin.Context) {
		reqCount := strconv.FormatInt(atomic.AddInt64(&requestCount, 1), 10)
		c.Set("RequestCount", reqCount)
		logger.Debugf("[%s] Incoming request: %s", reqCount, c.Request.URL.Path)
		start := time.Now()
		c.Next()

		msg := fmt.Sprintf("[%s] Request served", reqCount)
		status := c.Writer.Status()

		meta := []interface{}{
			"path", c.Request.URL.Path,
			"comment", c.Errors.ByType(gin.ErrorTypePrivate).String(),
			"latency", time.Now().Sub(start),
			"clientIP", c.ClientIP(),
			"method", c.Request.Method,
			"statusCode", status,
			"reqID", c.MustGet("RequestId"),
		}

		switch {
		case status == 200 || status == 201:
			logger.Infow(msg, meta...)
		case status == 404:
			logger.Warnw(msg, meta...)
		default:
			logger.Errorw(msg, meta...)
		}
	}
}

// getChartList fetches from the server and accumulates concurrent requests to be fulfilled all at once.
func (server *Server) getChartList(reqCount string) <-chan fetchedObjects {
	c := make(chan fetchedObjects, 1)

	server.fetchedObjectsLock.Lock()
	server.fetchedObjectsChans = append(server.fetchedObjectsChans, c)

	if len(server.fetchedObjectsChans) == 1 {
		// this unlock is wanted, while fetching the list, allow other channeled requests to be added
		server.fetchedObjectsLock.Unlock()

		objects, err := server.fetchChartsInStorage(reqCount)

		server.fetchedObjectsLock.Lock()
		// flush every other consumer that also wanted the index
		for _, foCh := range server.fetchedObjectsChans {
			foCh <- fetchedObjects{objects, err}
		}
		server.fetchedObjectsChans = nil
	}
	server.fetchedObjectsLock.Unlock()

	return c
}

func (server *Server) regenerateRepositoryIndex(diff storage.ObjectSliceDiff, storageObjects []storage.Object, reqCount string) <-chan indexRegeneration {
	c := make(chan indexRegeneration, 1)

	server.regenerationLock.Lock()
	server.regeneratedIndexesChans = append(server.regeneratedIndexesChans, c)

	if len(server.regeneratedIndexesChans) == 1 {
		server.regenerationLock.Unlock()

		index, err := server.regenerateRepositoryIndexWorker(diff, storageObjects, reqCount)

		server.regenerationLock.Lock()
		for _, riCh := range server.regeneratedIndexesChans {
			riCh <- indexRegeneration{index, err}
		}
		server.regeneratedIndexesChans = nil
	}
	server.regenerationLock.Unlock()

	return c
}

/*
syncRepositoryIndex is the workhorse of maintaining a coherent index cache. It is optimized for multiple requests
comming in a short period. When two requests for the backing store arrive, only the first is served, and other consumers receive the
result of this request. This allows very fast updates in constant time. See getChartList() and regenerateRepositoryIndex().
*/
func (server *Server) syncRepositoryIndex(reqCount string) (*repo.Index, error) {
	fo := <-server.getChartList(reqCount)

	if fo.err != nil {
		return nil, fo.err
	}

	diff := storage.GetObjectSliceDiff(server.StorageCache, fo.objects)

	// return fast if no changes
	if !diff.Change {
		return server.RepositoryIndex, nil
	}

	ir := <-server.regenerateRepositoryIndex(diff, fo.objects, reqCount)

	return ir.index, ir.err
}

func (server *Server) fetchChartsInStorage(reqCount string) ([]storage.Object, error) {
	server.Logger.Debugf("[%s] Fetching chart list from storage", reqCount)
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

func (server *Server) regenerateRepositoryIndexWorker(diff storage.ObjectSliceDiff, storageObjects []storage.Object, reqCount string) (*repo.Index, error) {
	server.Logger.Debugf("[%s] Regenerating index.yaml", reqCount)
	index := &repo.Index{
		IndexFile: server.RepositoryIndex.IndexFile,
		Raw:       server.RepositoryIndex.Raw,
		ChartURL:  server.RepositoryIndex.ChartURL,
	}

	for _, object := range diff.Removed {
		err := server.removeIndexObject(index, object, reqCount)
		if err != nil {
			return nil, err
		}
	}

	for _, object := range diff.Updated {
		err := server.updateIndexObject(index, object, reqCount)
		if err != nil {
			return nil, err
		}
	}

	// Parallelize retrieval of added objects to improve speed
	err := server.addIndexObjectsAsync(index, diff.Added, reqCount)
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

	server.Logger.Debugf("[%s] index.yaml regenerated", reqCount)
	return index, nil
}

func (server *Server) removeIndexObject(index *repo.Index, object storage.Object, reqCount string) error {
	chartVersion, err := server.getObjectChartVersion(object, false)
	if err != nil {
		return server.checkInvalidChartPackageError(object, err, "removed")
	}
	server.Logger.Debugw(fmt.Sprintf("[%s] Removing chart from index", reqCount),
		"name", chartVersion.Name,
		"version", chartVersion.Version,
	)
	index.RemoveEntry(chartVersion)
	return nil
}

func (server *Server) updateIndexObject(index *repo.Index, object storage.Object, reqCount string) error {
	chartVersion, err := server.getObjectChartVersion(object, true)
	if err != nil {
		return server.checkInvalidChartPackageError(object, err, "updated")
	}
	server.Logger.Debugw(fmt.Sprintf("[%s] Updating chart in index", reqCount),
		"name", chartVersion.Name,
		"version", chartVersion.Version,
	)
	index.UpdateEntry(chartVersion)
	return nil
}

func (server *Server) addIndexObjectsAsync(index *repo.Index, objects []storage.Object, reqCount string) error {
	numObjects := len(objects)
	if numObjects == 0 {
		return nil
	}

	server.Logger.Debugw(fmt.Sprintf("[%s] Loading charts packages from storage (this could take awhile)", reqCount),
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
					err = server.checkInvalidChartPackageError(o, err, "added")
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
		server.Logger.Debugw(fmt.Sprintf("[%s] Adding chart to index", reqCount),
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

func (server *Server) checkInvalidChartPackageError(object storage.Object, err error, action string) error {
	if err == repo.ErrorInvalidChartPackage {
		server.Logger.Warnw("Invalid package in storage",
			"action", action,
			"package", object.Path,
		)
		return nil
	}
	return err
}
