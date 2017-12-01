package storage

import (
	"errors"
	"io/ioutil"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
)

// AzureBlobBackend is a storage backend for Azure Blob Storage
type AzureBlobBackend struct {
	Container  *storage.Container
}

// NewAzureBlobBackend creates a new instance of AzureBlobBackend
func NewAzureBlobBackend(accountName string, accountKey string, containerName string) *AzureBlobBackend {
	client, err := storage.NewBasicClient(accountName, accountKey)

	var containerRef *storage.Container;

	if (err == nil) {
		blobClient := client.GetBlobService()
		containerRef = blobClient.GetContainerReference(containerName)
	}

	b := &AzureBlobBackend{
		Container: containerRef,
	}

	return b
}

// ListObjects lists all objects in Azure Blob Storage container
func (b AzureBlobBackend) ListObjects() ([]Object, error) {
	
	var objects []Object

	if (b.Container == nil) {
		return objects, errors.New("Unable to obtain a container reference.")
	}

	var params storage.ListBlobsParameters
	response, err := b.Container.ListBlobs(params)

	for _, blob := range response.Blobs {
		err = blob.GetProperties(nil);

		if (err != nil) {
			return objects, err;
		}

		object := Object {
			Path: blob.Name,
			Content: []byte{},
			LastModified: time.Time(blob.Properties.LastModified),
		}

		objects = append(objects, object)
	}
	return objects, nil
}

// GetObject retrieves an object from Azure Blob Storage, at path
func (b AzureBlobBackend) GetObject(path string) (Object, error) {
	var object Object
	object.Path = path
	var content []byte

	if (b.Container == nil) {
		return object, errors.New("Unable to obtain a container reference.")
	}

	blobReference := b.Container.GetBlobReference(path)
	exists, err := blobReference.Exists()

	if err != nil {
		return object, err
	}

	if !exists {
		return object, errors.New("Object does not exist.")
	}

	readCloser, err := blobReference.Get(nil)
	
	if err != nil {
		return object, err
	}
	
	// defer readCloser.Close()

	content, err = ioutil.ReadAll(readCloser)

	if err != nil {
		return object, err
	}
	object.Content = content
	err = blobReference.GetProperties(nil)
	object.LastModified = time.Time(blobReference.Properties.LastModified)
	return object, nil
}

// PutObject uploads an object to Azure Blob Storage container, at path
func (b AzureBlobBackend) PutObject(path string, content []byte) error {
	if (b.Container == nil) {
		return errors.New("Unable to obtain a container reference.")
	}

	blobReference := b.Container.GetBlobReference(path)

	err := blobReference.PutAppendBlob(nil)

	if (err == nil) {
		err = blobReference.AppendBlock(content, nil)
	}

	return err
}

// DeleteObject removes an object from Azure Blob Storage container, at path
func (b AzureBlobBackend) DeleteObject(path string) error {
	blobReference := b.Container.GetBlobReference(path)

	_, err := blobReference.DeleteIfExists(nil);

	return err
}
