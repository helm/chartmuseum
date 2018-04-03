package multitenant

import (
	pathutil "path/filepath"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "github.com/kubernetes-helm/chartmuseum/pkg/repo"

	helm_repo "k8s.io/helm/pkg/repo"
)

func (server *MultiTenantServer) getAllCharts(log cm_logger.LoggingFn, repo string) (map[string]helm_repo.ChartVersions, *HTTPError) {
	indexFile, err := server.getIndexFile(log, repo)
	if err != nil {
		return nil, &HTTPError{500, err.Message}
	}
	return indexFile.Entries, nil
}

func (server *MultiTenantServer) getChart(log cm_logger.LoggingFn, repo string, name string) (helm_repo.ChartVersions, *HTTPError) {
	allCharts, err := server.getAllCharts(log, repo)
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

func (server *MultiTenantServer) uploadChartPackage(log cm_logger.LoggingFn, repo string, content []byte) *HTTPError {
	filename, err := cm_repo.ChartPackageFilenameFromContent(content)
	if err != nil {
		return &HTTPError{500, err.Error()}
	}
	if !server.AllowOverwrite {
		_, err = server.StorageBackend.GetObject(pathutil.Join(repo, filename))
		if err == nil {
			return &HTTPError{409, "file already exists"}
		}
	}
	log(cm_logger.DebugLevel,"Adding package to storage",
		"package", filename,
	)
	err = server.StorageBackend.PutObject(pathutil.Join(repo, filename), content)
	if err != nil {
		return &HTTPError{500, err.Error()}
	}
	return nil
}

func (server *MultiTenantServer) uploadProvenanceFile(log cm_logger.LoggingFn, repo string, content []byte) *HTTPError {
	filename, err := cm_repo.ProvenanceFilenameFromContent(content)
	if err != nil {
		return &HTTPError{500, err.Error()}
	}
	if !server.AllowOverwrite {
		_, err = server.StorageBackend.GetObject(pathutil.Join(repo, filename))
		if err == nil {
			return &HTTPError{409, "file already exists"}
		}
	}
	log(cm_logger.DebugLevel,"Adding provenance file to storage",
		"provenance_file", filename,
	)
	err = server.StorageBackend.PutObject(pathutil.Join(repo, filename), content)
	if err != nil {
		return &HTTPError{500, err.Error()}
	}
	return nil
}
