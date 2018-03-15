package multitenant


import (
	"fmt"

	"github.com/kubernetes-helm/chartmuseum/pkg/repo"
)

func (server *MultiTenantServer) getIndexFile(prefix string) (*repo.Index, *HTTPError) {
	objects, err := server.StorageBackend.ListObjects(prefix)
	if err != nil {
		return new(repo.Index), &HTTPError{500, err.Error()}
	}

	indexFile := repo.NewIndex("")
	for _, object := range objects {
		op := object.Path
		objectPath := fmt.Sprintf("%s/%s", prefix, op)
		object, err = server.StorageBackend.GetObject(objectPath)
		if err != nil {
			continue
		}
		chartVersion, err := repo.ChartVersionFromStorageObject(object)
		if err != nil {
			continue
		}
		chartVersion.URLs[0] = fmt.Sprintf("charts/%s", op)
		indexFile.AddEntry(chartVersion)
	}

	indexFile.Regenerate()
	return indexFile, nil
}
