package chartmuseum

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

	"github.com/kubernetes-helm/chartmuseum/pkg/repo"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"

	"github.com/gin-gonic/gin"
	helm_repo "k8s.io/helm/pkg/repo"
)

type (
	fetchedObjects struct {
		objects []storage.Object
		err     error
	}
	indexRegeneration struct {
		index *repo.Index
		err   error
	}
)

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
