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
	"net/http"
	pathutil "path/filepath"
	"sort"
	"strings"

	"github.com/chartmuseum/storage"
	"github.com/gin-gonic/gin"
	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "helm.sh/chartmuseum/pkg/repo"

	"helm.sh/helm/v3/pkg/chart"
	helm_repo "helm.sh/helm/v3/pkg/repo"
)

func (server *MultiTenantServer) getAllCharts(log cm_logger.LoggingFn, repo string, offset int, limit int) (map[string]helm_repo.ChartVersions, *HTTPError) {
	indexFile, err := server.getIndexFile(log, repo)
	if err != nil {
		return nil, &HTTPError{http.StatusInternalServerError, err.Message}
	}
	if offset == 0 && limit == -1 {
		return indexFile.Entries, nil
	}
	result := map[string]helm_repo.ChartVersions{}
	var keys []string
	for k := range indexFile.Entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	end := offset + limit
	if len(keys) < end {
		end = len(keys)
	}
	for i := offset; i < end; i++ {
		result[keys[i]] = indexFile.Entries[keys[i]]
	}
	return result, nil
}

func (server *MultiTenantServer) getChart(log cm_logger.LoggingFn, repo string, name string) (helm_repo.ChartVersions, *HTTPError) {
	allCharts, err := server.getAllCharts(log, repo, 0, -1)
	if err != nil {
		return nil, err
	}
	chart := allCharts[name]
	if chart == nil {
		return nil, &HTTPError{http.StatusNotFound, "chart not found"}
	}
	return chart, nil
}

func (server *MultiTenantServer) getChartVersion(log cm_logger.LoggingFn, repo string, name string, version string) (*helm_repo.ChartVersion, *HTTPError) {
	indexFile, err := server.getIndexFile(log, repo)
	if err != nil {
		return nil, &HTTPError{http.StatusInternalServerError, err.Message}
	}
	if version == "latest" {
		version = ""
	}
	chartVersion, getErr := indexFile.Get(name, version)
	if getErr != nil {
		return nil, &HTTPError{http.StatusNotFound, getErr.Error()}
	}
	return chartVersion, nil
}

func (server *MultiTenantServer) deleteChartVersion(log cm_logger.LoggingFn, repo string, name string, version string) *HTTPError {
	filename := pathutil.Join(repo, cm_repo.ChartPackageFilenameFromNameVersion(name, version))
	log(cm_logger.DebugLevel, "Deleting package from storage",
		"package", filename,
	)
	deleteObjErr := server.StorageBackend.DeleteObject(filename)
	if deleteObjErr != nil {
		return &HTTPError{http.StatusNotFound, deleteObjErr.Error()}
	}
	provFilename := pathutil.Join(repo, cm_repo.ProvenanceFilenameFromNameVersion(name, version))
	server.StorageBackend.DeleteObject(provFilename) // ignore error here, may be no prov file
	return nil
}

func (server *MultiTenantServer) getChartFileName(log cm_logger.LoggingFn, repo string, name string, version string) (string, *HTTPError) {
	chartVersion, err := server.getChartVersion(log, repo, name, version)
	if err != nil {
		return "", err
	}
	if len(chartVersion.URLs) == 0 {
		return "", &HTTPError{http.StatusNotFound, "chart filename not found"}
	}
	split := strings.Split(chartVersion.URLs[0], "/")
	if len(split) < 2 {
		return "", &HTTPError{http.StatusNotFound, "chart filename not found"}
	}
	return split[1], nil
}

func (server *MultiTenantServer) uploadChartPackage(log cm_logger.LoggingFn, repo string, content []byte, force bool) (string, *HTTPError) {
	var filename string

	filename, err := cm_repo.ChartPackageFilenameFromContent(content)
	if err != nil {
		return filename, &HTTPError{http.StatusBadRequest, err.Error()}
	}

	if pathutil.Base(filename) != filename {
		// Name wants to break out of current directory
		return filename, &HTTPError{http.StatusBadRequest, fmt.Sprintf("%s is improperly formatted", filename)}
	}

	// we should ensure that whether chart is existed even if the `overwrite` option is set
	// For `overwrite` option , here will increase one `storage.GetObject` than before ; others should be equalvarant with the previous version.
	var found bool
	_, err = server.StorageBackend.GetObject(pathutil.Join(repo, filename))
	// found
	if err == nil {
		found = true
		// For those no-overwrite servers, return the Conflict error.
		if !server.AllowOverwrite && (!server.AllowForceOverwrite || !force) {
			return filename, &HTTPError{http.StatusConflict, "file already exists"}
		}
		// continue with the `overwrite` servers
	}

	limitReached, err := server.checkStorageLimit(repo, filename, force)
	if err != nil {
		return filename, &HTTPError{http.StatusInternalServerError, err.Error()}
	}
	if limitReached {
		return filename, &HTTPError{http.StatusInsufficientStorage, "repo has reached storage limit"}
	}
	log(cm_logger.DebugLevel, "Adding package to storage",
		"package", filename,
	)
	if err := server.PutWithLimit(&gin.Context{}, log, repo, filename, content); err != nil {
		return filename, &HTTPError{http.StatusInternalServerError, err.Error()}
	}
	if found {
		// here is a fake conflict error for outside call
		// In order to not add another return `bool` check (API Compatibility)
		return filename, &HTTPError{http.StatusConflict, ""}
	}
	return filename, nil
}

func (server *MultiTenantServer) uploadProvenanceFile(log cm_logger.LoggingFn, repo string, content []byte, force bool) *HTTPError {
	filename, err := cm_repo.ProvenanceFilenameFromContent(content)
	if err != nil {
		return &HTTPError{http.StatusInternalServerError, err.Error()}
	}

	if pathutil.Base(filename) != filename {
		// Name wants to break out of current directory
		return &HTTPError{http.StatusBadRequest, fmt.Sprintf("%s is improperly formatted", filename)}
	}

	if !server.AllowOverwrite && (!server.AllowForceOverwrite || !force) {
		_, err = server.StorageBackend.GetObject(pathutil.Join(repo, filename))
		if err == nil {
			return &HTTPError{http.StatusConflict, "file already exists"}
		}
	}
	limitReached, err := server.checkStorageLimit(repo, filename, force)
	if err != nil {
		return &HTTPError{http.StatusInternalServerError, err.Error()}
	}
	if limitReached {
		return &HTTPError{http.StatusInsufficientStorage, "repo has reached storage limit"}
	}
	log(cm_logger.DebugLevel, "Adding provenance file to storage",
		"provenance_file", filename,
	)
	err = server.StorageBackend.PutObject(pathutil.Join(repo, filename), content)
	if err != nil {
		return &HTTPError{http.StatusInternalServerError, err.Error()}
	}
	return nil
}

func (server *MultiTenantServer) checkStorageLimit(repo string, filename string, force bool) (bool, error) {
	if server.MaxStorageObjects > 0 {
		allObjects, err := server.StorageBackend.ListObjects(repo)
		if err != nil {
			return false, err
		}
		if len(allObjects) >= server.MaxStorageObjects {
			limitReached := true
			if server.AllowOverwrite || (server.AllowForceOverwrite && force) {
				// if the max has been reached, we should still allow
				// user to overwrite an existing file
				for _, object := range allObjects {
					if object.Path == filename {
						limitReached = false
						break
					}
				}
			}
			return limitReached, nil
		}
	}
	return false, nil
}

func extractFromChart(content []byte) (name string, version string, err error) {
	cv, err := cm_repo.ChartVersionFromStorageObject(storage.Object{
		Content: content,
	})
	if err != nil {
		return "", "", err
	}
	return cv.Metadata.Name, cv.Metadata.Version, nil
}

func (server *MultiTenantServer) PutWithLimit(ctx *gin.Context, log cm_logger.LoggingFn, repo string,
	filename string, content []byte) error {
	if server.ChartLimits == nil {
		log(cm_logger.DebugLevel, "PutWithLimit: per-chart-limit not set")
		return server.StorageBackend.PutObject(pathutil.Join(repo, filename), content)
	}
	limit := server.ChartLimits.Limit
	name, _, err := extractFromChart(content)
	if err != nil {
		return err
	}
	// lock the backend storage resource to always get the correct one
	server.ChartLimits.Lock()
	defer server.ChartLimits.Unlock()
	// clean the oldest chart(both index and storage)
	// storage cache first
	objs, err := server.StorageBackend.ListObjects(repo)
	if err != nil {
		return err
	}
	var newObjs []storage.Object
	for _, obj := range objs {
		if !strings.HasPrefix(obj.Path, name) || strings.HasSuffix(obj.Path, ".prov") {
			continue
		}
		log(cm_logger.DebugLevel, "PutWithLimit", "current object name", obj.Path)
		newObjs = append(newObjs, obj)
	}
	if len(newObjs) < limit {
		log(cm_logger.DebugLevel, "PutWithLimit", "current objects", len(newObjs))
		return server.StorageBackend.PutObject(pathutil.Join(repo, filename), content)
	}
	sort.Slice(newObjs, func(i, j int) bool {
		return newObjs[i].LastModified.Unix() < newObjs[j].LastModified.Unix()
	})

	log(cm_logger.DebugLevel, "PutWithLimit", "old chart", newObjs[0].Path)
	// should we support delete N out-of-date charts ?
	// and should we must ensure the delete operation is ok ?
	o, err := server.StorageBackend.GetObject(pathutil.Join(repo, newObjs[0].Path))
	if err != nil {
		return err
	}
	if err := server.StorageBackend.DeleteObject(pathutil.Join(repo, newObjs[0].Path)); err != nil {
		return fmt.Errorf("PutWithLimit: clean the old chart: %w", err)
	}
	cv, err := cm_repo.ChartVersionFromStorageObject(o)
	if err != nil {
		return fmt.Errorf("PutWithLimit: extract chartversion from storage object: %w", err)
	}
	if err = server.StorageBackend.PutObject(pathutil.Join(repo, filename), content); err != nil {
		return fmt.Errorf("PutWithLimit: put new chart: %w", err)
	}
	go server.emitEvent(ctx, repo, deleteChart, &helm_repo.ChartVersion{
		Metadata: &chart.Metadata{
			Name:    cv.Name,
			Version: cv.Version,
		},
		Removed: true,
	})
	return nil
}
