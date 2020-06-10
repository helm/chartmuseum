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
	"net/http"
	pathutil "path"
	"strings"

	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "helm.sh/chartmuseum/pkg/repo"

	"github.com/chartmuseum/storage"
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
		return nil, &HTTPError{http.StatusInternalServerError, "unsupported file extension"}
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
		return nil, &HTTPError{http.StatusNotFound, "object not found"}
	}

	var contentType string
	if isProvenanceFile {
		contentType = provenanceFileContentType
	} else {
		contentType = chartPackageContentType
	}

	storageObject := &StorageObject{
		Object:      &object,
		ContentType: contentType,
	}

	return storageObject, nil
}
