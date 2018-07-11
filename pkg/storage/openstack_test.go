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

type OpenstackTestSuite struct {
	suite.Suite
	BrokenOpenstackOSBackend   *OpenstackOSBackend
	NoPrefixOpenstackOSBackend *OpenstackOSBackend
}

func (suite *OpenstackTestSuite) SetupSuite() {
	osRegion := os.Getenv("TEST_STORAGE_OPENSTACK_REGION")

	backend := NewOpenstackOSBackend("fake-container-cant-exist-fbce123", "", osRegion, "")
	suite.BrokenOpenstackOSBackend = backend

	osContainer := os.Getenv("TEST_STORAGE_OPENSTACK_CONTAINER")
	backend = NewOpenstackOSBackend(osContainer, "", osRegion, "")
	suite.NoPrefixOpenstackOSBackend = backend

	data := []byte("some object")
	path := "deleteme.txt"

	err := suite.NoPrefixOpenstackOSBackend.PutObject(path, data)
	suite.Nil(err, "no error putting deleteme.txt using openstack backend")
}

func (suite *OpenstackTestSuite) TearDownSuite() {
	err := suite.NoPrefixOpenstackOSBackend.DeleteObject("deleteme.txt")
	suite.Nil(err, "no error deleting deleteme.txt using Openstack backend")
}

func (suite *OpenstackTestSuite) TestListObjects() {
	_, err := suite.BrokenOpenstackOSBackend.ListObjects("")
	suite.NotNil(err, "cannot list objects with bad container")

	_, err = suite.NoPrefixOpenstackOSBackend.ListObjects("")
	suite.Nil(err, "can list objects with good container, no prefix")
}

func (suite *OpenstackTestSuite) TestGetObject() {
	_, err := suite.BrokenOpenstackOSBackend.GetObject("this-file-cannot-possibly-exist.tgz")
	suite.NotNil(err, "cannot get objects with bad container")
}

func (suite *OpenstackTestSuite) TestPutObject() {
	err := suite.BrokenOpenstackOSBackend.PutObject("this-file-will-not-upload.txt", []byte{})
	suite.NotNil(err, "cannot put objects with bad container")
}

func TestOpenstackStorageTestSuite(t *testing.T) {
	if os.Getenv("TEST_CLOUD_STORAGE") == "1" &&
		os.Getenv("TEST_STORAGE_OPENSTACK_CONTAINER") != "" &&
		os.Getenv("TEST_STORAGE_OPENSTACK_REGION") != "" {
		suite.Run(t, new(OpenstackTestSuite))
	}
}
