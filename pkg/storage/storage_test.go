package storage

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type StorageTestSuite struct {
	suite.Suite
	StorageBackends map[string]Backend
	TempDirectory   string
}

func (suite *StorageTestSuite) setupStorageBackends() {
	timestamp := time.Now().Format("20060102150405")
	suite.TempDirectory = fmt.Sprintf("../../.test/storage-storage/%s", timestamp)
	suite.StorageBackends = make(map[string]Backend)
	suite.StorageBackends["LocalFilesystem"] = Backend(NewLocalFilesystemBackend(suite.TempDirectory))

	// create empty dir in local storage to make sure it doesnt end up in ListObjects
	err := os.MkdirAll(fmt.Sprintf("%s/%s", suite.TempDirectory, "ignoreme"), 0777)
	suite.Nil(err, "No error creating ignored dir in local storage")

	if os.Getenv("TEST_CLOUD_STORAGE") == "1" {
		prefix := fmt.Sprintf("unittest/%s", timestamp)
		s3Bucket := os.Getenv("TEST_STORAGE_AMAZON_BUCKET")
		s3Region := os.Getenv("TEST_STORAGE_AMAZON_REGION")
		gcsBucket := os.Getenv("TEST_STORAGE_GOOGLE_BUCKET")
		blobContainer := os.Getenv("TEST_STORAGE_MICROSOFT_CONTAINER")
		ossBucket := os.Getenv("TEST_STORAGE_ALIBABA_BUCKET")
		ossEndpoint := os.Getenv("TEST_STORAGE_ALIBABA_ENDPOINT")
		if s3Bucket != "" && s3Region != "" {
			suite.StorageBackends["AmazonS3"] = Backend(NewAmazonS3Backend(s3Bucket, prefix, s3Region, "", ""))
		}
		if gcsBucket != "" {
			suite.StorageBackends["GoogleCS"] = Backend(NewGoogleCSBackend(gcsBucket, prefix))
		}
		if blobContainer != "" {
			suite.StorageBackends["MicrosoftBlob"] = Backend(NewMicrosoftBlobBackend(blobContainer, prefix))
		}
		if ossBucket != "" {
			suite.StorageBackends["AlibabaCloudOSS"] = Backend(NewAlibabaCloudOSSBackend(ossBucket, prefix, ossEndpoint, ""))
		}
	}
}

func (suite *StorageTestSuite) SetupSuite() {
	suite.setupStorageBackends()

	for i := 1; i <= 9; i++ {
		data := []byte(fmt.Sprintf("test content %d", i))
		path := fmt.Sprintf("test%d.txt", i)
		for key, backend := range suite.StorageBackends {
			err := backend.PutObject(path, data)
			message := fmt.Sprintf("no error putting object %s using %s backend", path, key)
			suite.Nil(err, message)
		}
	}

	for key, backend := range suite.StorageBackends {
		if key == "LocalFilesystem" {
			continue
		}
		data := []byte("skipped object")
		path := "this/is/a/skipped/object.txt"
		err := backend.PutObject(path, data)
		message := fmt.Sprintf("no error putting skipped object %s using %s backend", path, key)
		suite.Nil(err, message)
	}
}

func (suite *StorageTestSuite) TearDownSuite() {
	defer os.RemoveAll(suite.TempDirectory)

	for i := 1; i <= 9; i++ {
		path := fmt.Sprintf("test%d.txt", i)
		for key, backend := range suite.StorageBackends {
			err := backend.DeleteObject(path)
			message := fmt.Sprintf("no error deleting object %s using %s backend", path, key)
			suite.Nil(err, message)
		}
	}

	for key, backend := range suite.StorageBackends {
		if key == "LocalFilesystem" {
			continue
		}
		path := "this/is/a/skipped/object.txt"
		err := backend.DeleteObject(path)
		message := fmt.Sprintf("no error deleting skipped object %s using %s backend", path, key)
		suite.Nil(err, message)
	}
}

func (suite *StorageTestSuite) TestListObjects() {
	for key, backend := range suite.StorageBackends {
		objects, err := backend.ListObjects("")
		message := fmt.Sprintf("no error listing objects using %s backend", key)
		suite.Nil(err, message)
		expectedNumObjects := 9
		message = fmt.Sprintf("%d objects listed using %s backend", expectedNumObjects, key)
		suite.Equal(expectedNumObjects, len(objects), message)
		for i, object := range objects {
			path := fmt.Sprintf("test%d.txt", (i + 1))
			message = fmt.Sprintf("object %s found in list objects using %s backend", path, key)
			suite.Equal(path, object.Path, message)
		}
	}
}

func (suite *StorageTestSuite) TestGetObject() {
	for key, backend := range suite.StorageBackends {
		for i := 1; i <= 9; i++ {
			path := fmt.Sprintf("test%d.txt", i)
			object, err := backend.GetObject(path)
			message := fmt.Sprintf("no error getting object %s using %s backend", path, key)
			suite.Nil(err, message)
			message = fmt.Sprintf("object %s content as expected using %s backend", path, key)
			suite.Equal(object.Content, []byte(fmt.Sprintf("test content %d", i)), message)
		}
	}
}

func (suite *StorageTestSuite) TestHasSuffix() {
	now := time.Now()
	o1 := Object{
		Path:         "mychart-0.1.0.tgz",
		Content:      []byte{},
		LastModified: now,
	}
	suite.True(o1.HasExtension("tgz"), "object has tgz suffix")
	o2 := Object{
		Path:         "mychart-0.1.0.txt",
		Content:      []byte{},
		LastModified: now,
	}
	suite.False(o2.HasExtension("tgz"), "object does not have tgz suffix")
}

func (suite *StorageTestSuite) TestGetObjectSliceDiff() {
	now := time.Now()
	os1 := []Object{
		{
			Path:         "test1.txt",
			Content:      []byte{},
			LastModified: now,
		},
	}
	os2 := []Object{}
	diff := GetObjectSliceDiff(os1, os2)
	suite.True(diff.Change, "change detected")
	suite.Equal(diff.Removed, os1, "removed slice populated")
	suite.Empty(diff.Added, "added slice empty")
	suite.Empty(diff.Updated, "updated slice empty")

	os2 = append(os2, os1[0])
	diff = GetObjectSliceDiff(os1, os2)
	suite.False(diff.Change, "no change detected")
	suite.Empty(diff.Removed, "removed slice empty")
	suite.Empty(diff.Added, "added slice empty")
	suite.Empty(diff.Updated, "updated slice empty")

	os2[0].LastModified = now.Add(1)
	diff = GetObjectSliceDiff(os1, os2)
	suite.True(diff.Change, "change detected")
	suite.Empty(diff.Removed, "removed slice empty")
	suite.Empty(diff.Added, "added slice empty")
	suite.Equal(diff.Updated, os2, "updated slice populated")

	os2[0].LastModified = now
	os2 = append(os2, Object{
		Path:         "test2.txt",
		Content:      []byte{},
		LastModified: now,
	})
	diff = GetObjectSliceDiff(os1, os2)
	suite.True(diff.Change, "change detected")
	suite.Empty(diff.Removed, "removed slice empty")
	suite.Equal(diff.Added, []Object{os2[1]}, "added slice empty")
	suite.Empty(diff.Updated, "updated slice empty")
}

func TestStorageTestSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}
