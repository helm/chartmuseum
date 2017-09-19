package repo

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"k8s.io/helm/pkg/proto/hapi/chart"
	helm_repo "k8s.io/helm/pkg/repo"
)

type IndexTestSuite struct {
	suite.Suite
	Index *Index
}

func getChartVersion(name string, patch int, created time.Time) *helm_repo.ChartVersion {
	version := fmt.Sprintf("1.0.%d", patch)
	metadata := chart.Metadata{
		Name:    name,
		Version: version,
	}
	chartVersion := helm_repo.ChartVersion{
		Metadata: &metadata,
		URLs:     []string{},
		Created:  created,
		Removed:  false,
		Digest:   "",
	}
	return &chartVersion
}

func (suite *IndexTestSuite) SetupSuite() {
	suite.Index = NewIndex()
	now := time.Now()
	for _, name := range []string{"a", "b", "c"} {
		for i := 0; i < 10; i++ {
			chartVersion := getChartVersion(name, i, now)
			suite.Index.AddEntry(chartVersion)
		}
	}
	chartVersion := getChartVersion("d", 0, now)
	suite.Index.AddEntry(chartVersion)
}

func (suite *IndexTestSuite) TestRegenerate() {
	err := suite.Index.Regenerate()
	suite.Nil(err)
}

func (suite *IndexTestSuite) TestUpdate() {
	now := time.Now()
	for _, name := range []string{"a", "b", "c"} {
		for i := 0; i < 5; i++ {
			chartVersion := getChartVersion(name, i, now)
			suite.Index.UpdateEntry(chartVersion)
		}
	}
}

func (suite *IndexTestSuite) TestRemove() {
	now := time.Now()
	for _, name := range []string{"a", "b", "c"} {
		for i := 5; i < 10; i++ {
			chartVersion := getChartVersion(name, i, now)
			suite.Index.RemoveEntry(chartVersion)
		}
	}
	chartVersion := getChartVersion("d", 0, now)
	suite.Index.RemoveEntry(chartVersion)
}

func TestIndexTestSuite(t *testing.T) {
	suite.Run(t, new(IndexTestSuite))
}
