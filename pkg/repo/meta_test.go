package repo

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type MetaTestSuite struct {
	suite.Suite
}

func (suite *MetaTestSuite) TestMetaFileFilenameFromContent() {
	goodContent := []byte(`{
"name": "mychart",
"version": "0.1.0",
"tests_passed": true
}`)

	badContentMalformedJSON := []byte(`{
"name": "mychart"
"version": "0.1.0",
"tests_passed": true
}`)

	badContentNoChartName := []byte(`{
"version": "0.1.0",
"tests_passed": true
}`)

	badContentNoChartVersion := []byte(`{
"name": "mychart"",
"tests_passed": true
}`)



	filename, err := MetaFilenameFromContent(goodContent)
	suite.Nil(err, "no error getting filename from good content")
	suite.Equal("mychart-0.1.0.tgz.meta", filename, "filename generated from good content")

	_, err = MetaFilenameFromContent(badContentMalformedJSON)
	suite.Equal(ErrorInvalidMetaFile, err, "ErrorInvalidMetaFile from bad content, Malformed json")

	_, err = MetaFilenameFromContent(badContentNoChartName)
	suite.Equal(ErrorInvalidMetaFile, err, "ErrorInvalidMetaFile from bad content, no name")

	_, err = MetaFilenameFromContent(badContentNoChartVersion)
	suite.Equal(ErrorInvalidMetaFile, err, "ErrorInvalidMetaFile from bad content, no version")
}

func TestMetaTestSuite(t *testing.T) {
	suite.Run(t, new(MetaTestSuite))
}
