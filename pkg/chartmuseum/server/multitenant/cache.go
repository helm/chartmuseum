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

/*
__../)
-._  \          /:
_.-'  \        ( \___
  ,'   \     .'`\.) :"-._______
,' .'/||    / /   _ `""`  ```  `,        _ _  ~ - _
 .' / ||   / |   ( `.~v~v~v~v~v'  _  -- *     *  _ -
'  /  ||  | .\    `. `.  __-  ~ -     ~         --   -
  /   ||  | :  `----`. `.  -~ _  _ ~ *           *  -
 /    ||   \:_     /  `. `.  - *__   -    -       __
/    .'/    `.`----\    `._;        --  _ *  -     _
     ||      `,_    `                     - -__ -
     ||       /  `---':
          HERE BE DRAGONS!!!
*/

import (
	"context"
	"encoding/json"
	"errors"
	pathutil "path"
	"sync"
	"time"

	"go.uber.org/zap"

	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "helm.sh/chartmuseum/pkg/repo"

	cm_storage "github.com/chartmuseum/storage"
	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	helm_repo "helm.sh/helm/v3/pkg/repo"
)

type (
	cacheEntry struct {
		// cryptic JSON field names to minimize size saved in cache
		RepoName  string         `json:"a"`
		RepoIndex *cm_repo.Index `json:"b"`
		RepoLock  sync.RWMutex
	}

	memoryCacheStore struct {
		cache sync.Map
	}

	event struct {
		Context      *gin.Context            `json:"-"`
		RepoName     string                  `json:"repo_name"`
		OpType       operationType           `json:"operation_type"`
		ChartVersion *helm_repo.ChartVersion `json:"chart_version"`
	}

	operationType int
)

const (
	updateChart operationType = 0
	addChart    operationType = 1
	deleteChart operationType = 2
)

var (
	EntrySavedMessage             = "Entry saved in cache store"
	CouldNotSaveEntryErrorMessage = "Could not save entry in cache store"
)

func (server *MultiTenantServer) primeCache() error {
	// only prime the cache if this is a single tenant setup
	if server.Router.Depth == 0 {
		log := server.Logger.ContextLoggingFn(&gin.Context{})
		_, err := server.getIndexFile(log, "")
		if err != nil {
			return errors.New(err.Message)
		}
	}
	return nil
}

// getChartList fetches from the server and accumulates concurrent requests to be fulfilled all at once.
func (server *MultiTenantServer) getChartList(log cm_logger.LoggingFn, repo string) <-chan fetchedObjects {
	ch := make(chan fetchedObjects, 1)
	tenant := server.Tenants[repo]

	tenant.FetchedObjectsLock.Lock()
	tenant.FetchedObjectsChans = append(tenant.FetchedObjectsChans, ch)

	if len(tenant.FetchedObjectsChans) == 1 {
		// this unlock is wanted, while fetching the list, allow other channeled requests to be added
		tenant.FetchedObjectsLock.Unlock()

		objects, err := server.fetchChartsInStorage(log, repo)

		tenant.FetchedObjectsLock.Lock()

		// flush every other consumer that also wanted the index
		for _, foCh := range tenant.FetchedObjectsChans {
			foCh <- fetchedObjects{objects, err}
		}

		tenant.FetchedObjectsChans = nil
	}

	tenant.FetchedObjectsLock.Unlock()

	return ch
}

func (server *MultiTenantServer) regenerateRepositoryIndex(log cm_logger.LoggingFn, entry *cacheEntry, diff cm_storage.ObjectSliceDiff) <-chan indexRegeneration {
	ch := make(chan indexRegeneration, 1)
	tenant := server.Tenants[entry.RepoName]

	tenant.RegeneratedIndexesChans = append(tenant.RegeneratedIndexesChans, ch)

	if len(tenant.RegeneratedIndexesChans) == 1 {
		index, err := server.regenerateRepositoryIndexWorker(log, entry, diff)
		for _, riCh := range tenant.RegeneratedIndexesChans {
			riCh <- indexRegeneration{index, err}
		}
		tenant.RegeneratedIndexesChans = nil
	}

	return ch
}

func (server *MultiTenantServer) regenerateRepositoryIndexWorker(log cm_logger.LoggingFn, entry *cacheEntry, diff cm_storage.ObjectSliceDiff) (*cm_repo.Index, error) {
	repo := entry.RepoName

	log(cm_logger.DebugLevel, "Regenerating index.yaml",
		"repo", repo,
	)
	index := &cm_repo.Index{
		IndexFile: entry.RepoIndex.IndexFile,
		RepoName:  repo,
		Raw:       entry.RepoIndex.Raw,
		ChartURL:  entry.RepoIndex.ChartURL,
		IndexLock: sync.RWMutex{},
	}

	for _, object := range diff.Removed {
		err := server.removeIndexObject(log, repo, index, object)
		if err != nil {
			return nil, err
		}
	}

	for _, object := range diff.Updated {
		err := server.updateIndexObject(log, repo, index, object)
		if err != nil {
			return nil, err
		}
	}

	// Parallelize retrieval of added objects to improve speed
	err := server.addIndexObjectsAsync(log, repo, index, diff.Added)
	if err != nil {
		return nil, err
	}

	err = index.Regenerate()
	if err != nil {
		return nil, err
	}

	log(cm_logger.DebugLevel, "index.yaml regenerated",
		"repo", repo,
	)

	entry.RepoIndex = index
	err = server.saveCacheEntry(log, entry)
	return index, err
}

func (server *MultiTenantServer) fetchChartsInStorage(log cm_logger.LoggingFn, repo string) ([]cm_storage.Object, error) {
	log(cm_logger.DebugLevel, "Fetching chart list from storage",
		"repo", repo,
	)
	allObjects, err := server.StorageBackend.ListObjects(repo)
	if err != nil {
		return []cm_storage.Object{}, err
	}

	// filter out storage objects that dont have extension used for chart packages (.tgz)
	filteredObjects := []cm_storage.Object{}
	for _, object := range allObjects {
		if object.HasExtension(cm_repo.ChartPackageFileExtension) {
			filteredObjects = append(filteredObjects, object)
		}
	}

	return filteredObjects, nil
}

func (server *MultiTenantServer) removeIndexObject(log cm_logger.LoggingFn, repo string, index *cm_repo.Index, object cm_storage.Object) error {
	chartVersion, err := server.getObjectChartVersion(repo, object, false)
	if err != nil {
		return server.checkInvalidChartPackageError(log, repo, object, err, "removed")
	}
	log(cm_logger.DebugLevel, "Removing chart from index",
		"repo", repo,
		"name", chartVersion.Name,
		"version", chartVersion.Version,
	)
	index.RemoveEntry(chartVersion)
	return nil
}

func (server *MultiTenantServer) updateIndexObject(log cm_logger.LoggingFn, repo string, index *cm_repo.Index, object cm_storage.Object) error {
	chartVersion, err := server.getObjectChartVersion(repo, object, true)
	if err != nil {
		return server.checkInvalidChartPackageError(log, repo, object, err, "updated")
	}
	log(cm_logger.DebugLevel, "Updating chart in index",
		"repo", repo,
		"name", chartVersion.Name,
		"version", chartVersion.Version,
	)
	index.UpdateEntry(chartVersion)
	return nil
}

func (server *MultiTenantServer) addIndexObjectsAsync(log cm_logger.LoggingFn, repo string, index *cm_repo.Index, objects []cm_storage.Object) error {
	numObjects := len(objects)
	if numObjects == 0 {
		return nil
	}

	log(cm_logger.DebugLevel, "Loading charts packages from storage (this could take awhile)",
		"repo", repo,
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
		go func(o cm_storage.Object) {
			if server.IndexLimit != 0 {
				// Limit parallelism to the index-limit parameter value
				// if there are more than IndexLimit concurrent fetches, this send will block
				server.Limiter <- struct{}{}
				// once work is over, read one Limiter channel item to allow other workers to continue
				defer func() { <-server.Limiter }()
			}
			select {
			case <-ctx.Done():
				return
			default:
				chartVersion, err := server.getObjectChartVersion(repo, o, true)
				if err != nil {
					err = server.checkInvalidChartPackageError(log, repo, o, err, "added")
					if err != nil {
						cancel()
					}
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
		log(cm_logger.DebugLevel, "Adding chart to index",
			"repo", repo,
			"name", cvRes.cv.Name,
			"version", cvRes.cv.Version,
		)
		index.AddEntry(cvRes.cv)
	}

	return nil
}

func (server *MultiTenantServer) getObjectChartVersion(repo string, object cm_storage.Object, load bool) (*helm_repo.ChartVersion, error) {
	op := object.Path
	if load {
		var err error
		objectPath := pathutil.Join(repo, op)
		object, err = server.StorageBackend.GetObject(objectPath)
		if err != nil {
			return nil, err
		}
		if len(object.Content) == 0 {
			return nil, cm_repo.ErrorInvalidChartPackage
		}
	}
	return cm_repo.ChartVersionFromStorageObject(object)
}

func (server *MultiTenantServer) checkInvalidChartPackageError(log cm_logger.LoggingFn, repo string, object cm_storage.Object, err error, action string) error {
	if err == cm_repo.ErrorInvalidChartPackage {
		log(cm_logger.WarnLevel, "Invalid package in storage",
			"repo", repo,
			"action", action,
			"package", object.Path,
		)
		return nil
	}
	return err
}

func (server *MultiTenantServer) initCacheEntry(log cm_logger.LoggingFn, repo string) (*cacheEntry, error) {
	var entry *cacheEntry
	var content []byte
	var err error

	server.TenantCacheKeyLock.Lock()
	defer server.TenantCacheKeyLock.Unlock()

	if _, ok := server.Tenants[repo]; !ok {
		server.Tenants[repo] = &tenantInternals{
			FetchedObjectsLock: &sync.Mutex{},
		}
	}

	if server.ExternalCacheStore == nil {
		var ok bool
		entry, ok = server.InternalCacheStore.Load(repo)
		if !ok {
			repoIndex := server.newRepositoryIndex(log, repo)
			entry = &cacheEntry{
				RepoName:  repo,
				RepoIndex: repoIndex,
				RepoLock:  sync.RWMutex{},
			}
			server.InternalCacheStore.Store(repo, entry)
		} else {
			log(cm_logger.DebugLevel, "Entry found in cache store",
				"repo", repo,
			)
		}
	} else {
		content, err = server.ExternalCacheStore.Get(repo)
		if err != nil {
			repoIndex := server.newRepositoryIndex(log, repo)
			entry = &cacheEntry{
				RepoName:  repo,
				RepoIndex: repoIndex,
				RepoLock:  sync.RWMutex{},
			}
			content, err = json.Marshal(entry)
			if err != nil {
				return nil, err
			}
			err := server.ExternalCacheStore.Set(repo, content)
			if err != nil {
				log(cm_logger.ErrorLevel, CouldNotSaveEntryErrorMessage,
					"error", err.Error(),
					"repo", repo,
				)
			}
			return entry, nil
		}

		log(cm_logger.DebugLevel, "Entry found in cache store",
			"repo", repo,
		)

		err = json.Unmarshal(content, &entry)
		if err != nil {
			return nil, err
		}
	}

	return entry, nil
}

func (server *MultiTenantServer) saveCacheEntry(log cm_logger.LoggingFn, entry *cacheEntry) error {
	repo := entry.RepoName
	if server.ExternalCacheStore == nil {
		server.InternalCacheStore.Store(repo, entry)
		log(cm_logger.DebugLevel, EntrySavedMessage,
			"repo", repo,
		)
	} else {
		content, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		err = server.ExternalCacheStore.Set(repo, content)
		if err != nil {
			log(cm_logger.ErrorLevel, CouldNotSaveEntryErrorMessage,
				"error", err.Error(),
				"repo", repo,
			)
		} else {
			log(cm_logger.DebugLevel, EntrySavedMessage,
				"repo", repo,
			)
		}
	}
	return nil
}

func (server *MultiTenantServer) newRepositoryIndex(log cm_logger.LoggingFn, repo string) *cm_repo.Index {
	var chartURL string
	if server.ChartURL != "" {
		chartURL = server.ChartURL
		if repo != "" {
			chartURL = chartURL + "/" + repo
		}
	}

	serverInfo := &cm_repo.ServerInfo{
		ContextPath: server.Router.ContextPath,
	}

	if !server.UseStatefiles {
		return cm_repo.NewIndex(chartURL, repo, serverInfo)
	}

	objectPath := pathutil.Join(repo, cm_repo.StatefileFilename)
	object, err := server.StorageBackend.GetObject(objectPath)
	if err != nil {
		return cm_repo.NewIndex(chartURL, repo, serverInfo)
	}

	indexFile := &cm_repo.IndexFile{}
	err = yaml.Unmarshal(object.Content, indexFile)
	if err != nil {
		log(cm_logger.WarnLevel, "index-cache.yaml found but could not be parsed",
			"repo", repo,
			"error", err.Error(),
		)
		return cm_repo.NewIndex(chartURL, repo, serverInfo)
	}

	log(cm_logger.DebugLevel, "index-cache.yaml loaded",
		"repo", repo,
	)

	return &cm_repo.Index{
		IndexFile: indexFile,
		RepoName:  repo,
		Raw:       object.Content,
		ChartURL:  chartURL,
		IndexLock: sync.RWMutex{},
	}
}

func (server *MultiTenantServer) initCacheTimer() {
	if server.CacheInterval > 0 {
		// delta update the cache every X duration
		// (in case the files on the disk are manually manipulated)
		go func() {
			t := time.NewTicker(server.CacheInterval)
			for _ = range t.C {
				server.rebuildIndex()
			}
		}()
	}
}

func (server *MultiTenantServer) emitEvent(c *gin.Context, repo string, operationType operationType, chart *helm_repo.ChartVersion) {
	server.EventChan <- event{
		Context:      c,
		RepoName:     repo,
		OpType:       operationType,
		ChartVersion: chart,
	}
}

func (server *MultiTenantServer) startEventListener() {
	server.Router.Logger.Debug("Starting internal event listener")
	for {

		e := <-server.EventChan
		log := server.Logger.ContextLoggingFn(e.Context)

		repo := e.RepoName
		log(cm_logger.DebugLevel, "Event received", zap.Any("event", e))

		entry, err := server.initCacheEntry(log, repo)
		if err != nil {
			log(cm_logger.ErrorLevel, "Error initializing cache entry", zap.Error(err), zap.String("repo", repo))
			continue
		}
		entry.RepoLock.RLock()
		index := entry.RepoIndex
		entry.RepoLock.RUnlock()

		server.TenantCacheKeyLock.Lock()
		_, ok := server.Tenants[e.RepoName]
		server.TenantCacheKeyLock.Unlock()

		if !ok {
			log(cm_logger.ErrorLevel, "Error find tenants repo name", zap.Error(err), zap.String("repo", repo))
			continue
		}

		if e.ChartVersion == nil {
			log(cm_logger.WarnLevel, "Event does not contain chart version", zap.String("repo", repo),
				"operation_type", e.OpType)
			continue
		}

		entry.RepoLock.Lock()
		switch e.OpType {
		case updateChart:
			index.UpdateEntry(e.ChartVersion)
		case addChart:
			index.AddEntry(e.ChartVersion)
		case deleteChart:
			index.RemoveEntry(e.ChartVersion)
		default:
			log(cm_logger.ErrorLevel, "Invalid operation type", zap.String("repo", repo),
				"operation_type", e.OpType)
			continue
		}

		err = index.Regenerate()
		if err != nil {
			log(cm_logger.ErrorLevel, "Error regenerating index", zap.Error(err), zap.String("repo", repo))
			continue
		}
		entry.RepoIndex = index
		entry.RepoLock.Unlock()
		err = server.saveCacheEntry(log, entry)
		if err != nil {
			log(cm_logger.ErrorLevel, "Error saving cache entry", zap.Error(err), zap.String("repo", repo))
			continue
		}

		if server.UseStatefiles {
			// Dont wait, save index-cache.yaml to storage in the background.
			// It is not crucial if this does not succeed, we will just log any errors
			go server.saveStatefile(log, e.RepoName, entry.RepoIndex.Raw)
		}

		log(cm_logger.DebugLevel, "Event handled successfully", zap.Any("event", e))
	}
}

func (server *MultiTenantServer) rebuildIndex() {
	if len(server.Tenants) == 0 {
		return
	}
	server.Logger.Info("Rebuilding index for all tenants in cache")
	for repo, _ := range server.Tenants {
		go server.rebuildIndexForTenant(repo)
	}
}

func (server *MultiTenantServer) rebuildIndexForTenant(repo string) {
	log := server.Logger.ContextLoggingFn(&gin.Context{})
	log(cm_logger.InfoLevel, "Rebuilding index for tenant", zap.String("repo", repo))
	entry, err := server.initCacheEntry(log, repo)
	if err != nil {
		errStr := err.Error()
		log(cm_logger.ErrorLevel, errStr,
			"repo", repo,
		)
		return
	}
	server.refreshCacheEntry(log, repo, entry)
}

func (server *MultiTenantServer) refreshCacheEntry(log cm_logger.LoggingFn, repo string, entry *cacheEntry) {
	fo := <-server.getChartList(log, repo)

	if fo.err != nil {
		errStr := fo.err.Error()
		log(cm_logger.ErrorLevel, errStr,
			"repo", repo,
		)
		return
	}
	entry.RepoLock.Lock()
	defer entry.RepoLock.Unlock()
	objects := server.getRepoObjectSlice(entry)
	diff := cm_storage.GetObjectSliceDiff(objects, fo.objects, server.TimestampTolerance)

	// return fast if no changes
	if !diff.Change {
		log(cm_logger.DebugLevel, "No change detected between cache and storage",
			"repo", repo,
		)
		return
	}

	log(cm_logger.DebugLevel, "Change detected between cache and storage",
		"repo", repo,
	)

	ir := <-server.regenerateRepositoryIndex(log, entry, diff)
	if ir.err != nil {
		errStr := ir.err.Error()
		log(cm_logger.ErrorLevel, errStr,
			"repo", repo,
		)
		return
	}
	entry.RepoIndex = ir.index
	if server.UseStatefiles {
		// Dont wait, save index-cache.yaml to storage in the background.
		// It is not crucial if this does not succeed, we will just log any errors
		go server.saveStatefile(log, repo, ir.index.Raw)
	}
}

func (m *memoryCacheStore) Load(key interface{}) (*cacheEntry, bool) {
	var entry *cacheEntry
	var okinterface bool
	value, ok := m.cache.Load(key)
	if ok {
		entry, okinterface = value.(*cacheEntry)
		if !okinterface {
			return nil, okinterface
		}
	}
	return entry, ok
}

func (m *memoryCacheStore) Store(key, value interface{}) {
	m.cache.Store(key, value)
}
