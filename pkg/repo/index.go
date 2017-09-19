package repo

import (
	"time"

	"github.com/ghodss/yaml"

	helm_repo "k8s.io/helm/pkg/repo"
)

var (
	// IndexFileContentType is the http content-type header for index.yaml
	IndexFileContentType = "application/x-yaml"
)

// Index represents the repository index (index.yaml)
type Index struct {
	*helm_repo.IndexFile
	Raw []byte
}

// NewIndex creates a new instance of Index
func NewIndex() *Index {
	index := Index{&helm_repo.IndexFile{}, []byte{}}
	index.Entries = map[string]helm_repo.ChartVersions{}
	index.APIVersion = helm_repo.APIVersionV1
	return &index
}

// Regenerate sorts entries in index file and sets current time for generated key
func (index *Index) Regenerate() error {
	index.SortEntries()
	index.Generated = time.Now().Round(time.Second)
	raw, err := yaml.Marshal(index.IndexFile)
	if err != nil {
		return err
	}
	index.Raw = raw
	return nil
}

// RemoveEntry removes a chart version from index
func (index *Index) RemoveEntry(chartVersion *helm_repo.ChartVersion) {
	for k := range index.Entries {
		if k == chartVersion.Name {
			for i, cv := range index.Entries[chartVersion.Name] {
				if cv.Version == chartVersion.Version {
					index.Entries[chartVersion.Name] = append(index.Entries[chartVersion.Name][:i],
						index.Entries[chartVersion.Name][i+1:]...)
					if len(index.Entries[chartVersion.Name]) == 0 {
						delete(index.Entries, chartVersion.Name)
					}
					break
				}
			}
			break
		}
	}
}

// AddEntry adds a chart version to index
func (index *Index) AddEntry(chartVersion *helm_repo.ChartVersion) {
	if _, ok := index.Entries[chartVersion.Name]; !ok {
		index.Entries[chartVersion.Name] = helm_repo.ChartVersions{}
	}
	index.Entries[chartVersion.Name] = append(index.Entries[chartVersion.Name], chartVersion)
}

// UpdateEntry updates a chart version in index
func (index *Index) UpdateEntry(chartVersion *helm_repo.ChartVersion) {
	for k := range index.Entries {
		if k == chartVersion.Name {
			for i, cv := range index.Entries[chartVersion.Name] {
				if cv.Version == chartVersion.Version {
					index.Entries[chartVersion.Name][i] = chartVersion
					break
				}
			}
			break
		}
	}
}
