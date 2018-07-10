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
	"strconv"
)

type AlibabaTestSuite struct {
	suite.Suite
	BrokenAlibabaOSSBackend   *AlibabaCloudOSSBackend
	NoPrefixAlibabaOSSBackend *AlibabaCloudOSSBackend
	SSEAlibabaOSSBackend      *AlibabaCloudOSSBackend
}

const testCount = 100

func (suite *AlibabaTestSuite) SetupSuite() {
	backend := NewAlibabaCloudOSSBackend("fake-bucket-cant-exist-fbce123", "", "", "")
	suite.BrokenAlibabaOSSBackend = backend

	ossBucket := os.Getenv("TEST_STORAGE_ALIBABA_BUCKET")
	ossEndpoint := os.Getenv("TEST_STORAGE_ALIBABA_ENDPOINT")
	backend = NewAlibabaCloudOSSBackend(ossBucket, "", ossEndpoint, "")
	suite.NoPrefixAlibabaOSSBackend = backend

	backend = NewAlibabaCloudOSSBackend(ossBucket, "ssetest", ossEndpoint, "AES256")
	suite.SSEAlibabaOSSBackend = backend

	data := []byte("some object")
	path := "deleteme.txt"

	for i := 0; i < testCount; i++ {
		newPath := strconv.Itoa(i) + path
		err := suite.NoPrefixAlibabaOSSBackend.PutObject(newPath, data)
		suite.Nil(err, "no error putting deleteme.txt using Alibaba Cloud OSS backend")

		err = suite.SSEAlibabaOSSBackend.PutObject(newPath, data)
		suite.Nil(err, "no error putting deleteme.txt using Alibaba Cloud OSS backend (SSE)")
	}
}

func (suite *AlibabaTestSuite) TearDownSuite() {
	path := "deleteme.txt"
	for i := 0; i < testCount; i++ {
		newPath := strconv.Itoa(i) + path

		err := suite.NoPrefixAlibabaOSSBackend.DeleteObject(newPath)
		suite.Nil(err, "no error deleting deleteme.txt using AlibabaOSS backend")

		err = suite.SSEAlibabaOSSBackend.DeleteObject(newPath)
		suite.Nil(err, "no error deleting deleteme.txt using AlibabaOSS backend")
	}
}

func (suite *AlibabaTestSuite) TestListObjects() {
	_, err := suite.BrokenAlibabaOSSBackend.ListObjects("")
	suite.NotNil(err, "cannot list objects with bad bucket")

	objs, err := suite.NoPrefixAlibabaOSSBackend.ListObjects("")
	suite.Nil(err, "can list objects with good bucket, no prefix")
	suite.Equal(len(objs), testCount, "able to list objects")

	objs, err = suite.SSEAlibabaOSSBackend.ListObjects("")
	suite.Nil(err, "can list objects with good bucket, SSE")
	suite.Equal(len(objs), testCount, "able to list objects")
}

func (suite *AlibabaTestSuite) TestGetObject() {
	_, err := suite.BrokenAlibabaOSSBackend.GetObject("this-file-cannot-possibly-exist.tgz")
	suite.NotNil(err, "cannot get objects with bad bucket")

	obj, err := suite.SSEAlibabaOSSBackend.GetObject("0deleteme.txt")
	suite.Equal([]byte("some object"), obj.Content, "able to get object with SSE")
}

func (suite *AlibabaTestSuite) TestPutObject() {
	err := suite.BrokenAlibabaOSSBackend.PutObject("this-file-will-not-upload.txt", []byte{})
	suite.NotNil(err, "cannot put objects with bad bucket")
}

func TestAlibabaStorageTestSuite(t *testing.T) {
	if os.Getenv("TEST_CLOUD_STORAGE") == "1" &&
		os.Getenv("TEST_STORAGE_ALIBABA_BUCKET") != "" &&
		os.Getenv("TEST_STORAGE_ALIBABA_ENDPOINT") != "" {
		suite.Run(t, new(AlibabaTestSuite))
	}
}
