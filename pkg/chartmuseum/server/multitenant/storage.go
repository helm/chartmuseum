package multitenant

import (
	"fmt"
	"strings"

	"github.com/kubernetes-helm/chartmuseum/pkg/repo"
	"github.com/kubernetes-helm/chartmuseum/pkg/storage"
)

var (
	chartPackageContentType = "application/x-tar"
	provenanceFileContentType = "application/pgp-signature"
)

type (
	StorageObject struct {
		*storage.Object
		ContentType string
	}
)

func (server *MultiTenantServer) getStorageObject(prefix string, filename string) (*StorageObject, *HTTPError) {
	isChartPackage := strings.HasSuffix(filename, repo.ChartPackageFileExtension)
	isProvenanceFile := strings.HasSuffix(filename, repo.ProvenanceFileExtension)
	if !isChartPackage && !isProvenanceFile {
		return nil, &HTTPError{500, "unsupported file extension"}
	}

	objectPath := fmt.Sprintf("%s/%s", prefix, filename)

	object, err := server.StorageBackend.GetObject(objectPath)
	if err != nil {
		return nil, &HTTPError{404, "object not found"}
	}

	var contentType string
	if isProvenanceFile {
		contentType = chartPackageContentType
	} else {
		contentType = chartPackageContentType
	}

	storageObject := &StorageObject{
		Object: &object,
		ContentType: contentType,
	}

	return storageObject, nil
}
