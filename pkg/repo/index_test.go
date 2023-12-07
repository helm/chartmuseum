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
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"strings"

	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/yaml"

	"helm.sh/helm/v3/pkg/chart"
	helm_repo "helm.sh/helm/v3/pkg/repo"
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
		URLs:     []string{fmt.Sprintf("charts/%s-%s.tgz", name, version)},
		Created:  created,
		Removed:  false,
		Digest:   "",
	}
	return &chartVersion
}

func (suite *IndexTestSuite) SetupSuite() {
	suite.Index = NewIndex("", "", &ServerInfo{}, false)
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
			suite.Empty(suite.Index.HasEntry(chartVersion))
		}
	}
	chartVersion := getChartVersion("d", 0, now)
	suite.Index.RemoveEntry(chartVersion)

	suite.Empty(suite.Index.HasEntry(chartVersion))
}

func (suite *IndexTestSuite) TestChartURLs() {
	index := NewIndex("", "", &ServerInfo{}, false)
	chartVersion := getChartVersion("a", 0, time.Now())
	index.AddEntry(chartVersion)
	suite.Equal("charts/a-1.0.0.tgz",
		index.Entries["a"][0].URLs[0], "relative chart url")

	index = NewIndex("http://mysite.com:8080", "", &ServerInfo{}, false)
	chartVersion = getChartVersion("a", 0, time.Now())
	index.AddEntry(chartVersion)
	suite.Equal("http://mysite.com:8080/charts/a-1.0.0.tgz",
		index.Entries["a"][0].URLs[0], "absolute chart url")
}

func (suite *IndexTestSuite) TestServerInfo() {
	serverInfo := &ServerInfo{}
	index := NewIndex("", "", serverInfo, false)
	suite.False(strings.Contains(string(index.Raw), "contextPath: /v1/helm"), "context path not in index")

	serverInfo = &ServerInfo{
		ContextPath: "/v1/helm",
	}
	index = NewIndex("", "", serverInfo, false)
	suite.True(strings.Contains(string(index.Raw), "contextPath: /v1/helm"), "context path is in index")
}

func (suite *IndexTestSuite) TestYAMLIndex() {
	index := NewIndex("", "", &ServerInfo{}, false)
	chartVersion := getChartVersion("a", 0, time.Now())
	index.AddEntry(chartVersion)
	suite.NoError(index.Regenerate())

	suite.False(json.Valid(index.Raw))
	suite.NoError(yaml.Unmarshal(index.Raw, &IndexFile{}))
}

func (suite *IndexTestSuite) TestJSONIndex() {
	index := NewIndex("", "", &ServerInfo{}, true)
	chartVersion := getChartVersion("a", 0, time.Now())
	index.AddEntry(chartVersion)
	suite.NoError(index.Regenerate())

	// Since YAML is a superset of JSON, any valid JSON should be valid YAML
	suite.True(json.Valid(index.Raw))
	suite.NoError(yaml.Unmarshal(index.Raw, &IndexFile{}))
}

func TestIndexTestSuite(t *testing.T) {
	suite.Run(t, new(IndexTestSuite))
}
