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
	cm_cache "github.com/kubernetes-helm/chartmuseum/pkg/cache"
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "github.com/kubernetes-helm/chartmuseum/pkg/repo"
	cm_storage "github.com/kubernetes-helm/chartmuseum/pkg/storage"
	pathutil "path"

	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	helm_repo "k8s.io/helm/pkg/repo"
)

type (
	cacheEntry struct {
		// cryptic JSON field names to minimize size saved in cache
		RepoName  string         `json:"a"`
		RepoIndex *cm_repo.Index `json:"b"`
		//FetchedObjectsLock      *sync.Mutex              `json:"c"`
		//FetchedObjectsChans     []chan fetchedObjects    `json:"d"`
		//RegenerationLock        *sync.Mutex              `json:"e"`
		//RegeneratedIndexesChans []chan indexRegeneration `json:"f"`
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

var (
	CouldNotSaveEntryErrorMessage = "Could not save entry in cache store due to insufficient memory allocation"
)

func (server *MultiTenantServer) primeCache() error {
	// only prime the cache if this is a single tenant setup
	if server.Router.Depth == 0 {
		log := server.Logger.ContextLoggingFn(&gin.Context{})
		_, err := server.getIndexFile(log, "")
		if err != nil {
			errStr := err.Message
			if errStr != cm_cache.ErrLargeEntry.Error() {
				return errors.New(errStr)
			}
		}
	}
	return nil
}

// getChartList fetches from the server and accumulates concurrent requests to be fulfilled all at once.
func (server *MultiTenantServer) getChartList(log cm_logger.LoggingFn, entry *cacheEntry) <-chan fetchedObjects {
	ch := make(chan fetchedObjects, 1)

	//entry.FetchedObjectsLock.Lock()
	//entry.FetchedObjectsChans = append(entry.FetchedObjectsChans, ch)

	//if len(entry.FetchedObjectsChans) == 1 {
	// this unlock is wanted, while fetching the list, allow other channeled requests to be added
	//entry.FetchedObjectsLock.Unlock()

	objects, err := server.fetchChartsInStorage(log, entry)
	ch <- fetchedObjects{objects, err}

	//entry.FetchedObjectsLock.Lock()

	// flush every other consumer that also wanted the index
	//for _, foCh := range entry.FetchedObjectsChans {
	//	foCh <- fetchedObjects{objects, err}
	//}
	//entry.FetchedObjectsChans = nil
	//}
	//entry.FetchedObjectsLock.Unlock()

	return ch
}

func (server *MultiTenantServer) regenerateRepositoryIndex(log cm_logger.LoggingFn, entry *cacheEntry, diff cm_storage.ObjectSliceDiff) <-chan indexRegeneration {
	ch := make(chan indexRegeneration, 1)

	//entry.RegenerationLock.Lock()
	//entry.RegeneratedIndexesChans = append(entry.RegeneratedIndexesChans, ch)

	//if len(entry.RegeneratedIndexesChans) == 1 {
	//	entry.RegenerationLock.Unlock()

	index, err := server.regenerateRepositoryIndexWorker(log, entry, diff)
	ch <- indexRegeneration{index, err}
	//	entry.RegenerationLock.Lock()
	//	for _, riCh := range entry.RegeneratedIndexesChans {
	//		riCh <- indexRegeneration{index, err}
	//	}
	//	entry.RegeneratedIndexesChans = nil
	//}
	//entry.RegenerationLock.Unlock()

	return ch
}

func (server *MultiTenantServer) regenerateRepositoryIndexWorker(log cm_logger.LoggingFn, entry *cacheEntry, diff cm_storage.ObjectSliceDiff) (*cm_repo.Index, error) {
	log(cm_logger.DebugLevel, "Regenerating index.yaml",
		"repo", entry.RepoName,
	)
	index := &cm_repo.Index{
		IndexFile: entry.RepoIndex.IndexFile,
		RepoName:  entry.RepoName,
		Raw:       entry.RepoIndex.Raw,
		ChartURL:  entry.RepoIndex.ChartURL,
	}

	for _, object := range diff.Removed {
		err := server.removeIndexObject(log, entry, index, object)
		if err != nil {
			return nil, err
		}
	}

	for _, object := range diff.Updated {
		err := server.updateIndexObject(log, entry, index, object)
		if err != nil {
			return nil, err
		}
	}

	// Parallelize retrieval of added objects to improve speed
	err := server.addIndexObjectsAsync(log, entry, index, diff.Added)
	if err != nil {
		return nil, err
	}

	err = index.Regenerate()
	if err != nil {
		return nil, err
	}

	entry.RepoIndex = index

	// save to cache
	content, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}
	err = server.CacheStore.Set(entry.RepoName, content)
	if err != nil {
		if err != cm_cache.ErrLargeEntry {
			return nil, err
		}
		log(cm_logger.WarnLevel, CouldNotSaveEntryErrorMessage,
			"repo", entry.RepoName,
		)
	}

	log(cm_logger.DebugLevel, "index.yaml regenerated",
		"repo", entry.RepoName,
	)
	return index, nil
}

func (server *MultiTenantServer) fetchChartsInStorage(log cm_logger.LoggingFn, entry *cacheEntry) ([]cm_storage.Object, error) {
	log(cm_logger.DebugLevel, "Fetching chart list from storage",
		"repo", entry.RepoName,
	)
	allObjects, err := server.StorageBackend.ListObjects(entry.RepoName)
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

func (server *MultiTenantServer) removeIndexObject(log cm_logger.LoggingFn, entry *cacheEntry, index *cm_repo.Index, object cm_storage.Object) error {
	chartVersion, err := server.getObjectChartVersion(entry, object, false)
	if err != nil {
		return server.checkInvalidChartPackageError(log, entry, object, err, "removed")
	}
	log(cm_logger.DebugLevel, "Removing chart from index",
		"repo", entry.RepoName,
		"name", chartVersion.Name,
		"version", chartVersion.Version,
	)
	index.RemoveEntry(chartVersion)
	return nil
}

func (server *MultiTenantServer) updateIndexObject(log cm_logger.LoggingFn, entry *cacheEntry, index *cm_repo.Index, object cm_storage.Object) error {
	chartVersion, err := server.getObjectChartVersion(entry, object, true)
	if err != nil {
		return server.checkInvalidChartPackageError(log, entry, object, err, "updated")
	}
	log(cm_logger.DebugLevel, "Updating chart in index",
		"repo", entry.RepoName,
		"name", chartVersion.Name,
		"version", chartVersion.Version,
	)
	index.UpdateEntry(chartVersion)
	return nil
}

func (server *MultiTenantServer) addIndexObjectsAsync(log cm_logger.LoggingFn, entry *cacheEntry, index *cm_repo.Index, objects []cm_storage.Object) error {
	numObjects := len(objects)
	if numObjects == 0 {
		return nil
	}

	log(cm_logger.DebugLevel, "Loading charts packages from storage (this could take awhile)",
		"repo", entry.RepoName,
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
				chartVersion, err := server.getObjectChartVersion(entry, o, true)
				if err != nil {
					err = server.checkInvalidChartPackageError(log, entry, o, err, "added")
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
			"repo", entry.RepoName,
			"name", cvRes.cv.Name,
			"version", cvRes.cv.Version,
		)
		index.AddEntry(cvRes.cv)
	}

	return nil
}

func (server *MultiTenantServer) getObjectChartVersion(entry *cacheEntry, object cm_storage.Object, load bool) (*helm_repo.ChartVersion, error) {
	op := object.Path
	if load {
		var err error
		objectPath := pathutil.Join(entry.RepoName, op)
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

func (server *MultiTenantServer) checkInvalidChartPackageError(log cm_logger.LoggingFn, entry *cacheEntry, object cm_storage.Object, err error, action string) error {
	if err == cm_repo.ErrorInvalidChartPackage {
		log(cm_logger.WarnLevel, "Invalid package in storage",
			"repo", entry.RepoName,
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

	server.IndexCacheKeyLock.Lock()
	defer server.IndexCacheKeyLock.Unlock()

	content, err = server.CacheStore.Get(repo)
	if err != nil {
		repoIndex := server.newRepositoryIndex(log, repo)
		entry = &cacheEntry{
			RepoName:  repo,
			RepoIndex: repoIndex,
			//FetchedObjectsLock: &sync.Mutex{},
			//RegenerationLock:   &sync.Mutex{},
		}
		content, err = json.Marshal(entry)
		if err != nil {
			return nil, err
		}
		err := server.CacheStore.Set(repo, content)
		if err != nil {
			if err != cm_cache.ErrLargeEntry {
				return nil, err
			}
			log(cm_logger.WarnLevel, CouldNotSaveEntryErrorMessage,
				"repo", entry.RepoName,
			)
		}
		return entry, nil
	}

	//fmt.Println(string(content))
	err = json.Unmarshal(content, &entry)
	if err != nil {
		//fmt.Printf("%+v\n", err)
		return nil, err
	}

	return entry, nil
}

func (server *MultiTenantServer) newRepositoryIndex(log cm_logger.LoggingFn, repo string) *cm_repo.Index {
	var chartURL string
	if server.ChartURL != "" {
		chartURL = server.ChartURL + "/" + repo
	}

	if !server.UseStatefiles {
		return cm_repo.NewIndex(chartURL, repo)
	}

	objectPath := pathutil.Join(repo, cm_repo.StatefileFilename)
	object, err := server.StorageBackend.GetObject(objectPath)
	if err != nil {
		return cm_repo.NewIndex(chartURL, repo)
	}

	indexFile := &helm_repo.IndexFile{}
	err = yaml.Unmarshal(object.Content, indexFile)
	if err != nil {
		log(cm_logger.WarnLevel, "index-cache.yaml found but could not be parsed",
			"repo", repo,
			"error", err.Error(),
		)
		return cm_repo.NewIndex(chartURL, repo)
	}

	log(cm_logger.DebugLevel, "index-cache.yaml loaded",
		"repo", repo,
	)

	return &cm_repo.Index{indexFile, repo, object.Content, chartURL}
}
