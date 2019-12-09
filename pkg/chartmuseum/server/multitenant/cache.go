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
	}
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

	tenant.RegenerationLock.Lock()
	tenant.RegeneratedIndexesChans = append(tenant.RegeneratedIndexesChans, ch)

	if len(tenant.RegeneratedIndexesChans) == 1 {
		tenant.RegenerationLock.Unlock()
		index, err := server.regenerateRepositoryIndexWorker(log, entry, diff)
		tenant.RegenerationLock.Lock()
		for _, riCh := range tenant.RegeneratedIndexesChans {
			riCh <- indexRegeneration{index, err}
		}
		tenant.RegeneratedIndexesChans = nil
	}

	tenant.RegenerationLock.Unlock()

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
			RegenerationLock:   &sync.Mutex{},
		}
	}

	if server.ExternalCacheStore == nil {
		var ok bool
		entry, ok = server.InternalCacheStore[repo]
		if !ok {
			repoIndex := server.newRepositoryIndex(log, repo)
			entry = &cacheEntry{
				RepoName:  repo,
				RepoIndex: repoIndex,
			}
			server.InternalCacheStore[repo] = entry
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
		server.InternalCacheStore[repo] = entry
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
	}
}
