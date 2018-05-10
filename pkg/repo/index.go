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
	// StatefileFilename is the cached index as it was last calculated
	StatefileFilename = "index-cache.yaml"
)

// Index represents the repository index (index.yaml)
type Index struct {
	*helm_repo.IndexFile
	Raw      []byte
	ChartURL string
	Repo     string
}

// NewIndex creates a new instance of Index
func NewIndex(chartURL, repo string) *Index {
	chartURL = strings.TrimSuffix(chartURL, "/")
	index := Index{&helm_repo.IndexFile{}, []byte{}, chartURL, repo}
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
	if entries, ok := index.Entries[chartVersion.Name]; ok {
		for i, cv := range entries {
			if cv.Version == chartVersion.Version {
				index.Entries[chartVersion.Name] = append(entries[:i],
					entries[i+1:]...)
				if len(index.Entries[chartVersion.Name]) == 0 {
					delete(index.Entries, chartVersion.Name)
				}
				break
			}
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

// HasEntry checks if index has already an entry
func (index *Index) HasEntry(chartVersion *helm_repo.ChartVersion) bool {
	if entries, ok := index.Entries[chartVersion.Name]; ok {
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
	if entries, ok := index.Entries[chartVersion.Name]; ok {
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
		chartVersion.URLs[0] = strings.Join([]string{index.ChartURL, chartVersion.URLs[0]}, "/")
	}
}

// UpdateMetrics updates chart index-related Prometheus metrics
func (index *Index) updateMetrics() {
	nChartVersions := 0
	for _, chartVersions := range index.Entries {
		nChartVersions += len(chartVersions)
	}
	chartTotalGaugeVec.WithLabelValues(index.Repo).Set(float64(len(index.Entries)))
	chartVersionTotalGaugeVec.WithLabelValues(index.Repo).Set(float64(nChartVersions))
}
