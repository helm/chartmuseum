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

	"github.com/Masterminds/semver/v3"
	"github.com/chartmuseum/storage"
	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "helm.sh/chartmuseum/pkg/repo"

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

func (server *MultiTenantServer) uploadChartPackage(log cm_logger.LoggingFn, repo string, content []byte, force bool) (string, *HTTPError) {
	filename, err := cm_repo.ChartPackageFilenameFromContent(content)
	if err != nil {
		return filename, &HTTPError{http.StatusInternalServerError, err.Error()}
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

	if server.EnforceSemver2 {
		version, err := cm_repo.ChartVersionFromStorageObject(storage.Object{
			Content: content,
			// Since we only need content to check for the chart version
			// left the others fields to be default
		})
		if err != nil {
			return filename, &HTTPError{http.StatusBadRequest, err.Error()}
		}
		if _, err := semver.StrictNewVersion(version.Metadata.Version); err != nil {
			return filename, &HTTPError{http.StatusBadRequest, fmt.Errorf("semver2 validation: %w", err).Error()}
		}
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
	err = server.StorageBackend.PutObject(pathutil.Join(repo, filename), content)
	if err != nil {
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

func (server *MultiTenantServer) uploadMetaFile(log cm_logger.LoggingFn, repo string, content []byte, force bool) *HTTPError {
	filename, err := cm_repo.MetaFilenameFromContent(content)
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
	log(cm_logger.DebugLevel, "Adding Meta file to storage",
		"meta_file", filename,
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
