package multitenant

import (
	pathutil "path"
	"strings"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "github.com/kubernetes-helm/chartmuseum/pkg/repo"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"
)

var (
	chartPackageContentType   = "application/x-tar"
	provenanceFileContentType = "application/pgp-signature"
)

type (
	StorageObject struct {
		*storage.Object
		ContentType string
	}
)

func (server *MultiTenantServer) getStorageObject(log cm_logger.LoggingFn, repo string, filename string) (*StorageObject, *HTTPError) {
	isChartPackage := strings.HasSuffix(filename, cm_repo.ChartPackageFileExtension)
	isProvenanceFile := strings.HasSuffix(filename, cm_repo.ProvenanceFileExtension)
	if !isChartPackage && !isProvenanceFile {
		log(cm_logger.WarnLevel, "unsupported file extension",
			"repo", repo,
			"filename", filename,
		)
		return nil, &HTTPError{500, "unsupported file extension"}
	}

	objectPath := pathutil.Join(repo, filename)

	object, err := server.StorageBackend.GetObject(objectPath)
	if err != nil {
		errStr := err.Error()
		log(cm_logger.WarnLevel, errStr,
			"repo", repo,
			"filename", filename,
		)
		// TODO determine if this is true 404
		return nil, &HTTPError{404, "object not found"}
	}

	var contentType string
	if isProvenanceFile {
		contentType = chartPackageContentType
	} else {
		contentType = chartPackageContentType
	}

	storageObject := &StorageObject{
		Object:      &object,
		ContentType: contentType,
	}

	return storageObject, nil
}
