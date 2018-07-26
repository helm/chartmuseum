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
	pathutil "path"

	cm_logger "github.com/helm/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "github.com/helm/chartmuseum/pkg/repo"
	cm_storage "github.com/helm/chartmuseum/pkg/storage"
)

var (
	indexFileContentType = "application/x-yaml"
)

func (server *MultiTenantServer) getIndexFile(log cm_logger.LoggingFn, repo string) (*cm_repo.Index, *HTTPError) {
	entry, err := server.initCacheEntry(log, repo)
	if err != nil {
		errStr := err.Error()
		log(cm_logger.ErrorLevel, errStr,
			"repo", repo,
		)
		return nil, &HTTPError{500, errStr}
	}

	fo := <-server.getChartList(log, repo)

	if fo.err != nil {
		errStr := fo.err.Error()
		log(cm_logger.ErrorLevel, errStr,
			"repo", repo,
		)
		return nil, &HTTPError{500, errStr}
	}

	objects := server.getRepoObjectSlice(entry)

	log(cm_logger.DebugLevel, "cache[0].time:", "", objects[0].LastModified)
	log(cm_logger.DebugLevel, "storage[0].time:", "", fo.objects[0].LastModified)
	log(cm_logger.DebugLevel, "equal?", "", objects[0].LastModified.Equal(fo.objects[0].LastModified))
	diff := cm_storage.GetObjectSliceDiff(objects, fo.objects)

	// return fast if no changes
	if !diff.Change {
		log(cm_logger.DebugLevel, "No change detected between cache and storage",
			"repo", repo,
		)
		return entry.RepoIndex, nil
	}

	log(cm_logger.DebugLevel, "Change detected between cache and storage",
		"repo", repo,
	)

	ir := <-server.regenerateRepositoryIndex(log, entry, diff)
	newRepoIndex := ir.index

	if ir.err != nil {
		errStr := ir.err.Error()
		log(cm_logger.ErrorLevel, errStr,
			"repo", repo,
		)
		return newRepoIndex, &HTTPError{500, errStr}
	}

	if server.UseStatefiles {
		// Dont wait, save index-cache.yaml to storage in the background.
		// It is not crucial if this does not succeed, we will just log any errors
		go server.saveStatefile(log, repo, ir.index.Raw)
	}

	return ir.index, nil
}

func (server *MultiTenantServer) saveStatefile(log cm_logger.LoggingFn, repo string, content []byte) {
	err := server.StorageBackend.PutObject(pathutil.Join(repo, cm_repo.StatefileFilename), content)
	if err != nil {
		log(cm_logger.WarnLevel, "Error saving index-cache.yaml",
			"repo", repo,
			"error", err.Error(),
		)
	}
	log(cm_logger.DebugLevel, "index-cache.yaml saved in storage",
		"repo", repo,
	)
}

func (server *MultiTenantServer) getRepoObjectSlice(entry *cacheEntry) []cm_storage.Object {
	var objects []cm_storage.Object
	for _, entry := range entry.RepoIndex.Entries {
		for _, chartVersion := range entry {
			object := cm_repo.StorageObjectFromChartVersion(chartVersion)
			objects = append(objects, object)
		}
	}
	// map ranging is out of order, so we have to sort it by alphabet
	alphabet := func(o1, o2 *cm_storage.Object) bool {
		return o1.Path < o2.Path
	}
	cm_storage.By(alphabet).Sort(objects)
	return objects
}
