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

package storage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type GoogleTestSuite struct {
	suite.Suite
	BrokenGoogleCSBackend   *GoogleCSBackend
	NoPrefixGoogleCSBackend *GoogleCSBackend
}

func (suite *GoogleTestSuite) SetupSuite() {
	backend := NewGoogleCSBackend("fake-bucket-cant-exist-fbce123", "")
	suite.BrokenGoogleCSBackend = backend

	gcsBucket := os.Getenv("TEST_STORAGE_GOOGLE_BUCKET")
	backend = NewGoogleCSBackend(gcsBucket, "")
	suite.NoPrefixGoogleCSBackend = backend

	data := []byte("some object")
	path := "deleteme.txt"
	err := suite.NoPrefixGoogleCSBackend.PutObject(path, data)
	suite.Nil(err, "no error putting deleteme.txt using GoogleCS backend")
}

func (suite *GoogleTestSuite) TearDownSuite() {
	err := suite.NoPrefixGoogleCSBackend.DeleteObject("deleteme.txt")
	suite.Nil(err, "no error deleting deleteme.txt using GoogleCS backend")
}

func (suite *GoogleTestSuite) TestListObjects() {
	_, err := suite.BrokenGoogleCSBackend.ListObjects("")
	suite.NotNil(err, "cannot list objects with bad bucket")

	_, err = suite.NoPrefixGoogleCSBackend.ListObjects("")
	suite.Nil(err, "can list objects with good bucket, no prefix")
}

func (suite *GoogleTestSuite) TestGetObject() {
	_, err := suite.BrokenGoogleCSBackend.GetObject("this-file-cannot-possibly-exist.tgz")
	suite.NotNil(err, "cannot get objects with bad bucket")
}

func (suite *GoogleTestSuite) TestPutObject() {
	err := suite.BrokenGoogleCSBackend.PutObject("this-file-will-not-upload.txt", []byte{})
	suite.NotNil(err, "cannot put objects with bad bucket")
}

func TestGoogleStorageTestSuite(t *testing.T) {
	if os.Getenv("TEST_CLOUD_STORAGE") == "1" &&
		os.Getenv("TEST_STORAGE_GOOGLE_BUCKET") != "" {
		suite.Run(t, new(GoogleTestSuite))
	}
}
