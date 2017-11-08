package repo

import (
	"bytes"
	"errors"
	"fmt"
	pathutil "path"
	"strings"

	"github.com/chartmuseum/chartmuseum/pkg/storage"

	"k8s.io/helm/pkg/chartutil"
	helm_chart "k8s.io/helm/pkg/proto/hapi/chart"
	helm_repo "k8s.io/helm/pkg/repo"
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
		URLs:     []string{fmt.Sprintf("charts/%s", object.Path)},
		Metadata: chart.Metadata,
		Digest:   digest,
		Created:  object.LastModified,
	}
	return chartVersion, nil
}

func chartFromContent(content []byte) (*helm_chart.Chart, error) {
	chart, err := chartutil.LoadArchive(bytes.NewBuffer(content))
	return chart, err
}

func emptyChartVersionFromPackageFilename(filename string) *helm_repo.ChartVersion {
	noExt := strings.TrimSuffix(pathutil.Base(filename), fmt.Sprintf(".%s", ChartPackageFileExtension))
	tmp := strings.Split(noExt, "-")
	lastIndex := len(tmp) - 1
	name := strings.Join(tmp[:lastIndex], "-")
	version := tmp[lastIndex]
	metadata := &helm_chart.Metadata{Name: name, Version: version}
	return &helm_repo.ChartVersion{Metadata: metadata}
}
