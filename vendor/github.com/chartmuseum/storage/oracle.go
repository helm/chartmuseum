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
	"bytes"
	"context"
	"encoding/binary"
	"io/ioutil"
	"os"
	pathutil "path"
	"time"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/common/auth"
	"github.com/oracle/oci-go-sdk/objectstorage"
)

// OracleCSBackend is a storage backend for Oracle Cloud Infrastructure Object Storage
type OracleCSBackend struct {
	Bucket        string
	Prefix        string
	Namespace     string
	CompartmentId string
	Client        objectstorage.ObjectStorageClient
	Context       context.Context
}

// NewOracleCSBackend creates a new instance of OracleCSBackend
func NewOracleCSBackend(bucket string, prefix string, region string, compartmentId string) *OracleCSBackend {

	var config common.ConfigurationProvider
	var err error

	authMethod := os.Getenv("ORACLE_AUTH_METHOD")

	if authMethod == "InstancePrincipal" {
		config, err = auth.InstancePrincipalConfigurationProvider()
		if err != nil {
			panic(err)
		}
	} else {
		config = common.DefaultConfigProvider()
	}

	c, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(config)

	if len(region) > 0 {
		c.SetRegion(region)
	}

	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	namespace, err := getNamespace(ctx, c)
	if err != nil {
		panic(err)
	}

	// Check if the bucket already exists
	request := objectstorage.GetBucketRequest{
		NamespaceName: &namespace,
		BucketName:    &bucket,
	}

	_, err = c.GetBucket(ctx, request)
	if err != nil {
		// Create the bucket if it does not exist
		_, err = createBucket(ctx, c, namespace, bucket, compartmentId)
		if err != nil {
			panic(err)
		}
	}

	prefix = cleanPrefix(prefix)
	b := &OracleCSBackend{
		Bucket:        bucket,
		Prefix:        prefix,
		Namespace:     namespace,
		CompartmentId: compartmentId,
		Client:        c,
		Context:       ctx,
	}
	return b
}

func createBucket(ctx context.Context, c objectstorage.ObjectStorageClient, namespace string, bucket string, compartmentId string) (string, error) {

	// Create the bucket
	request := objectstorage.CreateBucketRequest{
		NamespaceName: &namespace,
	}

	request.CompartmentId = &compartmentId
	request.Name = &bucket
	request.Metadata = make(map[string]string)
	request.PublicAccessType = objectstorage.CreateBucketDetailsPublicAccessTypeNopublicaccess
	_, err := c.CreateBucket(ctx, request)

	return bucket, err

}

func getNamespace(ctx context.Context, c objectstorage.ObjectStorageClient) (string, error) {
	var namespace string
	request := objectstorage.GetNamespaceRequest{}
	r, err := c.GetNamespace(ctx, request)
	if err != nil {
		return namespace, err
	}
	return *r.Value, nil
}

// ListObjects lists all objects in OCI Object Storage bucket, at prefix
func (b OracleCSBackend) ListObjects(prefix string) ([]Object, error) {
	var objects []Object
	prefix = pathutil.Join(b.Prefix, prefix)

	request := objectstorage.ListObjectsRequest{
		NamespaceName: &b.Namespace,
		BucketName:    &b.Bucket,
		Prefix:        &prefix,
	}

	rc, err := b.Client.ListObjects(b.Context, request)
	if err != nil {
		return objects, err
	}

	for i := 0; i < len(rc.ListObjects.Objects); i++ {
		attrs := rc.ListObjects.Objects[i]

		path := removePrefixFromObjectPath(prefix, *attrs.Name)
		if objectPathIsInvalid(path) {
			continue
		}

		var t time.Time
		if attrs.TimeCreated != nil {
			t = (*attrs.TimeCreated).Time
		}

		object := Object{
			Path:         path,
			Content:      []byte{},
			LastModified: t,
		}
		objects = append(objects, object)
	}
	return objects, nil
}

// GetObject retrieves an object from OCI Object Storage bucket, at prefix
func (b OracleCSBackend) GetObject(path string) (Object, error) {
	var object Object
	object.Path = path

	objectname := pathutil.Join(b.Prefix, path)

	request := objectstorage.GetObjectRequest{
		NamespaceName: &b.Namespace,
		BucketName:    &b.Bucket,
		ObjectName:    &objectname,
	}

	rc, err := b.Client.GetObject(b.Context, request)

	if err != nil {
		return object, err
	}

	object.LastModified = rc.LastModified.Time
	content, err := ioutil.ReadAll(rc.Content)

	if err != nil {
		return object, err
	}
	object.Content = content
	return object, nil
}

// PutObject uploads an object to OCI Object Storage bucket, at prefix
func (b OracleCSBackend) PutObject(path string, content []byte) error {

	objectname := pathutil.Join(b.Prefix, path)
	metadata := make(map[string]string)
	contentLen := int64(binary.Size(content))
	contentBody := ioutil.NopCloser(bytes.NewBuffer(content))

	request := objectstorage.PutObjectRequest{
		NamespaceName: &b.Namespace,
		BucketName:    &b.Bucket,
		ObjectName:    &objectname,
		PutObjectBody: contentBody,
		ContentLength: &contentLen,
		OpcMeta:       metadata,
	}

	_, err := b.Client.PutObject(b.Context, request)
	return err
}

// DeleteObject removes an object from OCI Object Storage bucket, at prefix
func (b OracleCSBackend) DeleteObject(path string) error {

	objectname := pathutil.Join(b.Prefix, path)

	request := objectstorage.DeleteObjectRequest{
		NamespaceName: &b.Namespace,
		BucketName:    &b.Bucket,
		ObjectName:    &objectname,
	}

	_, err := b.Client.DeleteObject(b.Context, request)
	return err
}
