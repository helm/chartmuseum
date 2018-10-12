/*
Copyright The Helm Authors.
Copyright (c) 2018, Oracle and/or its affiliates. All rights reserved.

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

type OracleTestSuite struct {
	suite.Suite
	BrokenOracleCSBackend   *OracleCSBackend
	NoPrefixOracleCSBackend *OracleCSBackend
}

func (suite *OracleTestSuite) SetupSuite() {
	backend := NewOracleCSBackend("fake-bucket-cant-exist-fbce123", "", "", "")
	suite.BrokenOracleCSBackend = backend

	ocsBucket := os.Getenv("TEST_STORAGE_ORACLE_BUCKET")
	ocsRegion := os.Getenv("TEST_STORAGE_ORACLE_REGION")
	ocsCompartmentId := os.Getenv("TEST_STORAGE_ORACLE_COMPARTMENTID")
	backend = NewOracleCSBackend(ocsBucket, "", ocsRegion, ocsCompartmentId)
	suite.NoPrefixOracleCSBackend = backend

	data := []byte("some object")
	path := "deleteme.txt"
	err := suite.NoPrefixOracleCSBackend.PutObject(path, data)
	suite.Nil(err, "no error putting deleteme.txt using OracleCS backend")
}

func (suite *OracleTestSuite) TearDownSuite() {
	err := suite.NoPrefixOracleCSBackend.DeleteObject("deleteme.txt")
	suite.Nil(err, "no error deleting deleteme.txt using OracleCS backend")
}

func (suite *OracleTestSuite) TestListObjects() {
	_, err := suite.BrokenOracleCSBackend.ListObjects("")
	suite.NotNil(err, "cannot list objects with bad bucket")

	_, err = suite.NoPrefixOracleCSBackend.ListObjects("")
	suite.Nil(err, "can list objects with good bucket, no prefix")
}

func (suite *OracleTestSuite) TestGetObject() {
	_, err := suite.BrokenOracleCSBackend.GetObject("this-file-cannot-possibly-exist.tgz")
	suite.NotNil(err, "cannot get objects with bad bucket")
}

func (suite *OracleTestSuite) TestPutObject() {
	err := suite.BrokenOracleCSBackend.PutObject("this-file-will-not-upload.txt", []byte{})
	suite.NotNil(err, "cannot put objects with bad bucket")
}

func TestOracleStorageTestSuite(t *testing.T) {
	if os.Getenv("TEST_CLOUD_STORAGE") == "1" &&
		os.Getenv("TEST_STORAGE_ORACLE_BUCKET") != "" {
		suite.Run(t, new(OracleTestSuite))
	}
}
