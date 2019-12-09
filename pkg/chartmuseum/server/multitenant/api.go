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
	pathutil "path/filepath"
	"sort"

	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "helm.sh/chartmuseum/pkg/repo"

	helm_repo "helm.sh/helm/v3/pkg/repo"
)

func (server *MultiTenantServer) getAllCharts(log cm_logger.LoggingFn, repo string, offset int, limit int) (map[string]helm_repo.ChartVersions, *HTTPError) {
	indexFile, err := server.getIndexFile(log, repo)
	if err != nil {
		return nil, &HTTPError{500, err.Message}
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
	end := offset+limit
	if len(keys) < end {
		end = len(keys)
	}
	for i:=offset; i < end ; i++ {
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
		return nil, &HTTPError{404, "chart not found"}
	}
	return chart, nil
}

func (server *MultiTenantServer) getChartVersion(log cm_logger.LoggingFn, repo string, name string, version string) (*helm_repo.ChartVersion, *HTTPError) {
	indexFile, err := server.getIndexFile(log, repo)
	if err != nil {
		return nil, &HTTPError{500, err.Message}
	}
	if version == "latest" {
		version = ""
	}
	chartVersion, getErr := indexFile.Get(name, version)
	if getErr != nil {
		return nil, &HTTPError{404, getErr.Error()}
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
		return &HTTPError{404, deleteObjErr.Error()}
	}
	provFilename := pathutil.Join(repo, cm_repo.ProvenanceFilenameFromNameVersion(name, version))
	server.StorageBackend.DeleteObject(provFilename) // ignore error here, may be no prov file
	return nil
}

func (server *MultiTenantServer) uploadChartPackage(log cm_logger.LoggingFn, repo string, content []byte, force bool) *HTTPError {
	filename, err := cm_repo.ChartPackageFilenameFromContent(content)
	if err != nil {
		return &HTTPError{500, err.Error()}
	}

	if pathutil.Base(filename) != filename {
		// Name wants to break out of current directory
		return &HTTPError{400, fmt.Sprintf("%s is improperly formatted", filename)}
	}

	if !server.AllowOverwrite && (!server.AllowForceOverwrite || !force) {
		_, err = server.StorageBackend.GetObject(pathutil.Join(repo, filename))
		if err == nil {
			return &HTTPError{409, "file already exists"}
		}
	}
	limitReached, err := server.checkStorageLimit(repo, filename, force)
	if err != nil {
		return &HTTPError{500, err.Error()}
	}
	if limitReached {
		return &HTTPError{507, "repo has reached storage limit"}
	}
	log(cm_logger.DebugLevel, "Adding package to storage",
		"package", filename,
	)
	err = server.StorageBackend.PutObject(pathutil.Join(repo, filename), content)
	if err != nil {
		return &HTTPError{500, err.Error()}
	}
	return nil
}

func (server *MultiTenantServer) uploadProvenanceFile(log cm_logger.LoggingFn, repo string, content []byte, force bool) *HTTPError {
	filename, err := cm_repo.ProvenanceFilenameFromContent(content)
	if err != nil {
		return &HTTPError{500, err.Error()}
	}

	if pathutil.Base(filename) != filename {
		// Name wants to break out of current directory
		return &HTTPError{400, fmt.Sprintf("%s is improperly formatted", filename)}
	}

	if !server.AllowOverwrite && (!server.AllowForceOverwrite || !force) {
		_, err = server.StorageBackend.GetObject(pathutil.Join(repo, filename))
		if err == nil {
			return &HTTPError{409, "file already exists"}
		}
	}
	limitReached, err := server.checkStorageLimit(repo, filename, force)
	if err != nil {
		return &HTTPError{500, err.Error()}
	}
	if limitReached {
		return &HTTPError{507, "repo has reached storage limit"}
	}
	log(cm_logger.DebugLevel, "Adding provenance file to storage",
		"provenance_file", filename,
	)
	err = server.StorageBackend.PutObject(pathutil.Join(repo, filename), content)
	if err != nil {
		return &HTTPError{500, err.Error()}
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
