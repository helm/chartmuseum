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

package repo

import (
	"fmt"
	"strings"
	"time"

	"github.com/ghodss/yaml"

	helm_repo "helm.sh/helm/v3/pkg/repo"
)

var (
	// IndexFileContentType is the http content-type header for index.yaml
	IndexFileContentType = "application/x-yaml"
	StatefileFilename    = "index-cache.yaml"
)

type (
	// ServerInfo contains extra data about the server
	ServerInfo struct {
		ContextPath string `json:"contextPath,omitempty"`
	}

	// IndexFile is a copy of Helm struct with extra data
	IndexFile struct {
		*helm_repo.IndexFile
		ServerInfo *ServerInfo `json:"serverInfo"`
	}

	// Index represents the repository index (index.yaml)
	Index struct {
		// cryptic JSON field names to minimize size saved in cache
		*IndexFile `json:"a"`
		RepoName   string `json:"b"`
		Raw        []byte `json:"c"`
		ChartURL   string `json:"d"`
	}
)

// NewIndex creates a new instance of Index
func NewIndex(chartURL string, repo string, serverInfo *ServerInfo) *Index {
	indexFile := &IndexFile{
		IndexFile:  &helm_repo.IndexFile{},
		ServerInfo: serverInfo,
	}
	index := Index{indexFile, repo, []byte{}, chartURL}
	index.Entries = map[string]helm_repo.ChartVersions{}
	index.APIVersion = helm_repo.APIVersionV1
	index.Regenerate()
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
	index.updateMetrics()
	return nil
}

// RemoveEntry removes a chart version from index
func (index *Index) RemoveEntry(chartVersion *helm_repo.ChartVersion) {
	path := chartObjectPath(chartVersion)
	index.remove(path)
}

// RemoveEntryWithObjectPath removes chart version from index (with the chart object path)
func (index *Index) RemoveEntryWithObjectPath(path string) {
	index.remove(path)
}

func (index *Index) remove(path string) {
	if entries, ok := index.Entries[path]; ok {
		for i, cv := range entries {
			// version is always located at the tail of the path
			if strings.HasSuffix(path, cv.Version) {
				index.Entries[path] = append(entries[:i],
					entries[i+1:]...)
				if len(index.Entries[path]) == 0 {
					delete(index.Entries, path)
				}
				break
			}
		}
	}
}

// AddEntry adds a chart version to index
func (index *Index) AddEntry(chartVersion *helm_repo.ChartVersion) {
	path := chartObjectPath(chartVersion)
	if _, ok := index.Entries[path]; !ok {
		index.Entries[path] = helm_repo.ChartVersions{}
	}
	index.setChartURL(chartVersion)
	index.Entries[path] = append(index.Entries[path], chartVersion)
}

// HasEntry checks if index has already an entry
func (index *Index) HasEntry(chartVersion *helm_repo.ChartVersion) bool {
	if entries, ok := index.Entries[chartObjectPath(chartVersion)]; ok {
		for _, cv := range entries {
			if cv.Version == chartVersion.Version {
				return true
			}
		}
	}
	return false
}

// UpdateEntry updates a chart version in index
func (index *Index) UpdateEntry(chartVersion *helm_repo.ChartVersion) {
	if entries, ok := index.Entries[chartObjectPath(chartVersion)]; ok {
		for i, cv := range entries {
			if cv.Version == chartVersion.Version {
				index.setChartURL(chartVersion)
				entries[i] = chartVersion
				break
			}
		}
	}
}

func (index *Index) setChartURL(chartVersion *helm_repo.ChartVersion) {
	if index.ChartURL != "" {
		chartVersion.URLs[0] = index.ChartURL + "/" + chartVersion.URLs[0]
	}
}

// UpdateMetrics updates chart index-related Prometheus metrics
func (index *Index) updateMetrics() {
	nChartVersions := 0
	for _, chartVersions := range index.Entries {
		nChartVersions += len(chartVersions)
	}
	chartTotalGaugeVec.WithLabelValues(index.RepoName).Set(float64(len(index.Entries)))
	chartVersionTotalGaugeVec.WithLabelValues(index.RepoName).Set(float64(nChartVersions))
}

// chartObjectPath will be used as the key to access the index entries.
// We can not easily use the chartVersion's Name as the key
// because we can not parse the chart filename with the standard chartName and chartVersion properly(so much PRs and FIXs to the current parse confliction)
// With no semantic version chart , it even be worse to manage the chart cache (Remove).
// After we change the entries key from ChartName to ChartPath ,we can easily delete the cache with the object.Path
// and no need to parse the object.Path to ChartName and ChartVersion with lots of effort.
func chartObjectPath(cv *helm_repo.ChartVersion) string {
	return objectPath(cv.Name, cv.Version)
}

func objectPath(name, version string) string {
	return fmt.Sprintf("%s-%s", name, version)
}
