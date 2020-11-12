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
	"bytes"
	"errors"
	"fmt"
	pathutil "path"
	"strconv"
	"strings"

	"github.com/chartmuseum/storage"
	helm_chart "helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	helm_repo "helm.sh/helm/v3/pkg/repo"
)

var (
	// ChartPackageFileExtension is the file extension used for chart packages
	ChartPackageFileExtension = "tgz"

	// ChartPackageContentType is the http content-type header for chart packages
	ChartPackageContentType = "application/x-tar"

	// ErrorInvalidChartPackage is raised when a chart package is invalid
	ErrorInvalidChartPackage = errors.New("invalid chart package")
)

// ChartPackageFilenameFromNameVersion returns a chart filename from a name and version
func ChartPackageFilenameFromNameVersion(name string, version string) string {
	filename := fmt.Sprintf("%s-%s.%s", name, version, ChartPackageFileExtension)
	return filename
}

// ChartPackageFilenameFromContent returns a chart filename from binary content
func ChartPackageFilenameFromContent(content []byte) (string, error) {
	chart, err := chartFromContent(content)
	if err != nil {
		return "", err
	}
	meta := chart.Metadata
	filename := fmt.Sprintf("%s-%s.%s", meta.Name, meta.Version, ChartPackageFileExtension)
	return filename, nil
}

// ChartVersionFromStorageObject returns a chart version from a storage object
func ChartVersionFromStorageObject(object storage.Object) (*helm_repo.ChartVersion, error) {
	if len(object.Content) == 0 {
		if object.Meta.Version != "" && object.Meta.Name != "" {
			return &helm_repo.ChartVersion{
				Metadata: &helm_chart.Metadata{Name: object.Meta.Name, Version: object.Meta.Version},
			}, nil
		}
		chartVersion := emptyChartVersionFromPackageFilename(object.Path)
		if chartVersion.Name == "" || chartVersion.Version == "" {
			return nil, ErrorInvalidChartPackage
		}
		return chartVersion, nil
	}
	chart, err := chartFromContent(object.Content)
	if err != nil {
		return nil, ErrorInvalidChartPackage
	}
	digest, err := provenanceDigestFromContent(object.Content)
	if err != nil {
		return nil, err
	}
	chartVersion := &helm_repo.ChartVersion{
		URLs:     []string{fmt.Sprintf("charts/%s", pathutil.Base(object.Path))},
		Metadata: chart.Metadata,
		Digest:   digest,
		Created:  object.LastModified,
	}
	return chartVersion, nil
}

// StorageObjectFromChartVersion returns a storage object from a chart version (empty content)
func StorageObjectFromChartVersion(chartVersion *helm_repo.ChartVersion) storage.Object {
	meta := storage.Metadata{}
	if chartVersion.Metadata != nil {
		meta.Name = chartVersion.Name
		meta.Version = chartVersion.Version
	}
	object := storage.Object{
		Meta:         meta,
		Path:         pathutil.Base(chartVersion.URLs[0]),
		Content:      []byte{},
		LastModified: chartVersion.Created,
	}
	return object
}

func chartFromContent(content []byte) (*helm_chart.Chart, error) {
	chart, err := loader.LoadArchive(bytes.NewBuffer(content))
	return chart, err
}

func emptyChartVersionFromPackageFilename(filename string) *helm_repo.ChartVersion {
	noExt := strings.TrimSuffix(pathutil.Base(filename), fmt.Sprintf(".%s", ChartPackageFileExtension))
	parts := strings.Split(noExt, "-")
	lastIndex := len(parts) - 1
	name := parts[0]
	version := ""

	for idx := lastIndex; idx >= 1; idx-- {
		if _, err := strconv.Atoi(string(parts[idx][0])); err == nil { // see if this part looks like a version (starts w int)
			version = strings.Join(parts[idx:], "-")
			name = strings.Join(parts[:idx], "-")
			break
		}
	}
	if version == "" { // no parts looked like a real version, just take everything after last hyphen
		name = strings.Join(parts[:lastIndex], "-")
		version = parts[lastIndex]
	}

	metadata := &helm_chart.Metadata{Name: name, Version: version}
	return &helm_repo.ChartVersion{Metadata: metadata}
}
