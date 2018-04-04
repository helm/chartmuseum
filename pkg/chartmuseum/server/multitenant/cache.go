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
	"errors"
	pathutil "path"
	"sync"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "github.com/kubernetes-helm/chartmuseum/pkg/repo"
	cm_storage "github.com/kubernetes-helm/chartmuseum/pkg/storage"

	helm_repo "k8s.io/helm/pkg/repo"
	"github.com/gin-gonic/gin"
)

type (
	cachedIndexFile struct {
		RepositoryIndex         *cm_repo.Index
		StorageCache            []cm_storage.Object
		fetchedObjectsLock      *sync.Mutex
		fetchedObjectsChans     []chan fetchedObjects
		regenerationLock        *sync.Mutex
		regeneratedIndexesChans []chan indexRegeneration
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

	server.IndexCache[repo].fetchedObjectsLock.Lock()
	server.IndexCache[repo].fetchedObjectsChans = append(server.IndexCache[repo].fetchedObjectsChans, ch)

	if len(server.IndexCache[repo].fetchedObjectsChans) == 1 {
		// this unlock is wanted, while fetching the list, allow other channeled requests to be added
		server.IndexCache[repo].fetchedObjectsLock.Unlock()

		objects, err := server.fetchChartsInStorage(log, repo)

		server.IndexCache[repo].fetchedObjectsLock.Lock()

		// flush every other consumer that also wanted the index
		for _, foCh := range server.IndexCache[repo].fetchedObjectsChans {
			foCh <- fetchedObjects{objects, err}
		}
		server.IndexCache[repo].fetchedObjectsChans = nil
	}
	server.IndexCache[repo].fetchedObjectsLock.Unlock()

	return ch
}

func (server *MultiTenantServer) regenerateRepositoryIndex(log cm_logger.LoggingFn, repo string, diff cm_storage.ObjectSliceDiff, storageObjects []cm_storage.Object) <-chan indexRegeneration {
	ch := make(chan indexRegeneration, 1)

	server.IndexCache[repo].regenerationLock.Lock()
	server.IndexCache[repo].regeneratedIndexesChans = append(server.IndexCache[repo].regeneratedIndexesChans, ch)

	if len(server.IndexCache[repo].regeneratedIndexesChans) == 1 {
		server.IndexCache[repo].regenerationLock.Unlock()

		index, err := server.regenerateRepositoryIndexWorker(log, repo, diff, storageObjects)

		server.IndexCache[repo].regenerationLock.Lock()
		for _, riCh := range server.IndexCache[repo].regeneratedIndexesChans {
			riCh <- indexRegeneration{index, err}
		}
		server.IndexCache[repo].regeneratedIndexesChans = nil
	}
	server.IndexCache[repo].regenerationLock.Unlock()

	return ch
}

func (server *MultiTenantServer) regenerateRepositoryIndexWorker(log cm_logger.LoggingFn, repo string, diff cm_storage.ObjectSliceDiff, storageObjects []cm_storage.Object) (*cm_repo.Index, error) {
	log(cm_logger.DebugLevel, "Regenerating index.yaml",
		"repo", repo,
	)
	index := &cm_repo.Index{
		IndexFile: server.IndexCache[repo].RepositoryIndex.IndexFile,
		Raw:       server.IndexCache[repo].RepositoryIndex.Raw,
		ChartURL:  server.IndexCache[repo].RepositoryIndex.ChartURL,
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

	// It is very important that these two stay in sync as they reflect the same reality. StorageCache serves
	// as object modification time cache, and RepositoryIndex is the canonical cached index.
	server.IndexCache[repo].RepositoryIndex = index
	server.IndexCache[repo].StorageCache = storageObjects

	log(cm_logger.DebugLevel, "index.yaml regenerated",
		"repo", repo,
	)
	return index, nil
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

func (server *MultiTenantServer) initCachedIndexFile(log cm_logger.LoggingFn, repo string) {
	server.IndexCacheKeyLock.Lock()
	defer server.IndexCacheKeyLock.Unlock()
	if _, ok := server.IndexCache[repo]; !ok {
		var chartURL string
		if server.ChartURL != "" {
			chartURL = server.ChartURL + "/" + repo
		}
		server.IndexCache[repo] = &cachedIndexFile{
			RepositoryIndex:    cm_repo.NewIndex(chartURL),
			StorageCache:       []cm_storage.Object{},
			fetchedObjectsLock: &sync.Mutex{},
			regenerationLock:   &sync.Mutex{},
		}
	}
}
