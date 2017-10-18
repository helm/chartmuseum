package repo

import (
	"strings"
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
	Raw      []byte
	ChartURL string
}

// NewIndex creates a new instance of Index
func NewIndex(chartURL string) *Index {
	chartURL = strings.TrimSuffix(chartURL, "/")
	index := Index{&helm_repo.IndexFile{}, []byte{}, chartURL}
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
	index.updateMetrics()
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
	index.setChartURL(chartVersion)
	index.Entries[chartVersion.Name] = append(index.Entries[chartVersion.Name], chartVersion)
}

// UpdateEntry updates a chart version in index
func (index *Index) UpdateEntry(chartVersion *helm_repo.ChartVersion) {
	for k := range index.Entries {
		if k == chartVersion.Name {
			for i, cv := range index.Entries[chartVersion.Name] {
				if cv.Version == chartVersion.Version {
					index.setChartURL(chartVersion)
					index.Entries[chartVersion.Name][i] = chartVersion
					break
				}
			}
			break
		}
	}
}

func (index *Index) setChartURL(chartVersion *helm_repo.ChartVersion) {
	if index.ChartURL != "" {
		chartVersion.URLs[0] = strings.Join([]string{index.ChartURL, chartVersion.URLs[0]}, "/")
	}
}

// UpdateMetrics updates chart index-related Prometheus metrics
func (index *Index) updateMetrics() {
	nChartVersions := 0
	for _, chartVersions := range index.Entries {
		nChartVersions += len(chartVersions)
	}
	chartTotalGauge.Set(float64(len(index.Entries)))
	chartVersionTotalGauge.Set(float64(nChartVersions))
}
