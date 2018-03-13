package storage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MicrosoftTestSuite struct {
	suite.Suite
	BrokenAzureBlobBackend   *MicrosoftBlobBackend
	NoPrefixAzureBlobBackend *MicrosoftBlobBackend
}

func (suite *MicrosoftTestSuite) SetupSuite() {
	backend := NewMicrosoftBlobBackend("fake-container-cant-exist-fbce123", "")
	suite.BrokenAzureBlobBackend = backend

	containerName := os.Getenv("TEST_STORAGE_MICROSOFT_CONTAINER")
	backend = NewMicrosoftBlobBackend(containerName, "")
	suite.NoPrefixAzureBlobBackend = backend

	data := []byte("some object")
	path := "deleteme.txt"
	err := suite.NoPrefixAzureBlobBackend.PutObject(path, data)
	suite.Nil(err, "no error putting deleteme.txt using Azure backend")
}

func (suite *MicrosoftTestSuite) TearDownSuite() {
	err := suite.NoPrefixAzureBlobBackend.DeleteObject("deleteme.txt")
	suite.Nil(err, "no error deleting deleteme.txt using Azure backend")
}

func (suite *MicrosoftTestSuite) TestListObjects() {
	_, err := suite.BrokenAzureBlobBackend.ListObjects("")
	suite.NotNil(err, "cannot list objects with bad bucket")

	_, err = suite.NoPrefixAzureBlobBackend.ListObjects("")
	suite.Nil(err, "can list objects with good bucket, no prefix")
}

func (suite *MicrosoftTestSuite) TestGetObject() {
	_, err := suite.BrokenAzureBlobBackend.GetObject("this-file-cannot-possibly-exist.tgz")
	suite.NotNil(err, "cannot get objects with bad bucket")
}

func (suite *MicrosoftTestSuite) TestPutObject() {
	err := suite.BrokenAzureBlobBackend.PutObject("this-file-will-not-upload.txt", []byte{})
	suite.NotNil(err, "cannot put objects with bad bucket")
}

func TestAzureStorageTestSuite(t *testing.T) {
	if os.Getenv("TEST_CLOUD_STORAGE") == "1" &&
		os.Getenv("TEST_STORAGE_AZURE_CONTAINER") != "" {
		suite.Run(t, new(MicrosoftTestSuite))
	}
}
