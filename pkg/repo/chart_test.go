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
	"io/ioutil"
	"testing"
	"time"

	"github.com/chartmuseum/storage"
	"github.com/stretchr/testify/suite"
	"helm.sh/helm/v3/pkg/chart"
	helm_repo "helm.sh/helm/v3/pkg/repo"
)

type ChartTestSuite struct {
	suite.Suite
	TarballContent []byte
}

func (suite *ChartTestSuite) SetupSuite() {
	tarballPath := "../../testdata/charts/mychart/mychart-0.1.0.tgz"
	content, err := ioutil.ReadFile(tarballPath)
	suite.Nil(err, "no error reading test tarball")
	suite.TarballContent = content
}

func (suite *ChartTestSuite) TestChartPackageFilenameFromNameVersion() {
	filename := ChartPackageFilenameFromNameVersion("mychart", "2.3.4")
	suite.Equal("mychart-2.3.4.tgz", filename, "filename as expected")
}

func (suite *ChartTestSuite) TestChartVersionFromStorageObject() {
	object := storage.Object{
		Path:         "mychart-2.3.4.tgz",
		Content:      []byte{},
		LastModified: time.Now(),
	}
	chartVersion, err := ChartVersionFromStorageObject(object)
	suite.Nil(err, "no error creating ChartVersion from storage.Object")
	suite.Equal("mychart", chartVersion.Name, "chart name as expected")
	suite.Equal("2.3.4", chartVersion.Version, "chart version as expected")

	object.Content = suite.TarballContent
	chartVersion, err = ChartVersionFromStorageObject(object)
	suite.Nil(err)
	suite.Equal("mychart", chartVersion.Name, "chart name as expected")
	suite.Equal("0.1.0", chartVersion.Version, "chart version as expected")

	object.Content = []byte("this should create an error")
	_, err = ChartVersionFromStorageObject(object)
	suite.NotNil(err, "error creating ChartVersion from storage.Object with bad content")

	brokenObject := storage.Object{
		Path:         "brokenchart.tgz",
		Content:      []byte{},
		LastModified: time.Now(),
	}
	_, err = ChartVersionFromStorageObject(brokenObject)
	suite.Equal(err, ErrorInvalidChartPackage, "error creating ChartVersion from storage.Object with bad content")

	// Issue #22
	snapshotObject := storage.Object{
		Path:         "mychart-1.0.4-SNAPSHOT.tgz",
		Content:      []byte{},
		LastModified: time.Now(),
	}
	chartVersion, err = ChartVersionFromStorageObject(snapshotObject)
	suite.Nil(err)
	suite.Equal("mychart", chartVersion.Name, "chart name as expected")
	suite.Equal("1.0.4-SNAPSHOT", chartVersion.Version, "chart version as expected")

	snapshotObject2 := storage.Object{
		Meta:         storage.Metadata{Name: "mychart", Version: "1.0.4-SNAPSHOT-1"},
		Path:         "mychart-1.0.4-SNAPSHOT-1.tgz",
		Content:      []byte{},
		LastModified: time.Now(),
	}
	chartVersion, err = ChartVersionFromStorageObject(snapshotObject2)
	suite.Nil(err)
	suite.Equal("mychart", chartVersion.Name, "chart name as expected")
	suite.Equal("1.0.4-SNAPSHOT-1", chartVersion.Version, "chart version as expected")

	snapshotObject3 := storage.Object{
		Meta:         storage.Metadata{Name: "mychart", Version: "1.0-SNAPSHOT-1"},
		Path:         "mychart-1.0-SNAPSHOT-1.tgz",
		Content:      []byte{},
		LastModified: time.Now(),
	}
	chartVersion, err = ChartVersionFromStorageObject(snapshotObject3)
	suite.Nil(err)
	suite.Equal("mychart", chartVersion.Name, "chart name as expected")
	suite.Equal("1.0-SNAPSHOT-1", chartVersion.Version, "chart version as expected")

	snapshotObject4 := storage.Object{
		Meta:         storage.Metadata{Name: "mychart", Version: "1-SNAPSHOT-1"},
		Path:         "mychart-1-SNAPSHOT-1.tgz",
		Content:      []byte{},
		LastModified: time.Now(),
	}
	chartVersion, err = ChartVersionFromStorageObject(snapshotObject4)
	suite.Nil(err)
	suite.Equal("mychart", chartVersion.Name, "chart name as expected")
	suite.Equal("1-SNAPSHOT-1", chartVersion.Version, "chart version as expected")

	multiHyphenObject := storage.Object{
		Path:         "my-long-hyphenated-chart-name-1.0.4.tgz",
		Content:      []byte{},
		LastModified: time.Now(),
	}
	chartVersion, err = ChartVersionFromStorageObject(multiHyphenObject)
	suite.Nil(err)
	suite.Equal("my-long-hyphenated-chart-name", chartVersion.Name, "chart name as expected")
	suite.Equal("1.0.4", chartVersion.Version, "chart version as expected")

	multiHyphenSnapshotObject := storage.Object{
		Path:         "my-long-hyphenated-chart-name-1.0.4-SNAPSHOT.tgz",
		Content:      []byte{},
		LastModified: time.Now(),
	}
	chartVersion, err = ChartVersionFromStorageObject(multiHyphenSnapshotObject)
	suite.Nil(err)
	suite.Equal("my-long-hyphenated-chart-name", chartVersion.Name, "chart name as expected")
	suite.Equal("1.0.4-SNAPSHOT", chartVersion.Version, "chart version as expected")

	crapVersionObject := storage.Object{
		Path:         "my-long-hyphenated-chart-name-crapversion.tgz",
		Content:      []byte{},
		LastModified: time.Now(),
	}
	chartVersion, err = ChartVersionFromStorageObject(crapVersionObject)
	suite.Nil(err)
	suite.Equal("my-long-hyphenated-chart-name", chartVersion.Name, "chart name as expected")
	suite.Equal("crapversion", chartVersion.Version, "chart version as expected")

	// Issue #261
	hyphenDigitalObject := storage.Object{
		Path:         "mychart-1x-1.0.4.tgz",
		Content:      []byte{},
		LastModified: time.Now(),
	}
	chartVersion, err = ChartVersionFromStorageObject(hyphenDigitalObject)
	suite.Nil(err)
	suite.Equal("mychart-1x", chartVersion.Name, "chart name as expected")
	suite.Equal("1.0.4", chartVersion.Version, "chart version as expected")

	hyphenDigitalObject.Path = "mychart-1x-1.0.4-rc1.tgz"
	chartVersion, err = ChartVersionFromStorageObject(hyphenDigitalObject)
	suite.Nil(err)
	suite.Equal("mychart-1x", chartVersion.Name, "chart name as expected")
	suite.Equal("1.0.4-rc1", chartVersion.Version, "chart version as expected")

	hyphenDigitalObject.Path = "mychart-1x-1.0.4-rc1-SNAPSHOT.tgz"
	chartVersion, err = ChartVersionFromStorageObject(hyphenDigitalObject)
	suite.Nil(err)
	suite.Equal("mychart-1x", chartVersion.Name, "chart name as expected")
	suite.Equal("1.0.4-rc1-SNAPSHOT", chartVersion.Version, "chart version as expected")
}

func (suite *ChartTestSuite) TestStorageObjectFromChartVersion() {
	now := time.Now()
	chartVersion := &helm_repo.ChartVersion{
		Metadata: &chart.Metadata{
			Name:    "mychart",
			Version: "0.1.0",
		},
		URLs:    []string{"charts/mychart-0.1.0.tgz"},
		Created: now,
	}
	object := StorageObjectFromChartVersion(chartVersion)
	suite.Equal(now, object.LastModified, "object last modified as expected")
	suite.Equal("mychart-0.1.0.tgz", object.Path, "object path as expected")
	suite.Equal("mychart", object.Meta.Name, "object chart name as expected")
	suite.Equal("0.1.0", object.Meta.Version, "object chart version as expected")
	suite.Equal([]byte{}, object.Content, "object content as expected")
}

func (suite *ChartTestSuite) TestChartPackageFilenameFromContent() {
	filename, err := ChartPackageFilenameFromContent([]byte{})
	suite.NotNil(err, "error getting tarball filename with empty byte array")
	suite.Equal("", filename, "filename blank with empty byte array")

	filename, err = ChartPackageFilenameFromContent(suite.TarballContent)
	suite.Nil(err, "no error getting filename from test tarball content")
	suite.Equal("mychart-0.1.0.tgz", filename, "chart tarball filename as expected")
}

func TestChartTestSuite(t *testing.T) {
	suite.Run(t, new(ChartTestSuite))
}
