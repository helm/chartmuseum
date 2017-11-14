package repo

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/kubernetes-helm/chartmuseum/pkg/storage"

	"github.com/stretchr/testify/suite"
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
