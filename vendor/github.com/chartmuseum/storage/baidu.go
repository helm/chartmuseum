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
	"io/ioutil"
	"os"
	pathutil "path"
	"time"

	"github.com/baidubce/bce-sdk-go/services/bos"
	"github.com/baidubce/bce-sdk-go/services/bos/api"
)

// BaiduBOSBackend is a storage backend for Baidu Cloud BOS
type BaiduBOSBackend struct {
	Client *bos.Client
	Bucket string
	Prefix string
}

// NewBaiduBOSBackend creates a new instance of BaiduBOSBackend
func NewBaiDuBOSBackend(bucket string, prefix string, endpoint string) *BaiduBOSBackend {
	accessKeyId := os.Getenv("BAIDU_CLOUD_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("BAIDU_CLOUD_ACCESS_KEY_SECRET")

	if len(accessKeyId) == 0 {
		panic("BAIDU_CLOUD_ACCESS_KEY_ID environment variable is not set")
	}

	if len(accessKeySecret) == 0 {
		panic("BAIDU_CLOUD_ACCESS_KEY_SECRET environment variable is not set")
	}

	if len(endpoint) == 0 {
		// Set default endpoint
		endpoint = "bj.bcebos.com"
	}

	client, err := bos.NewClient(accessKeyId, accessKeySecret, endpoint)

	if err != nil {
		panic("Failed to create BOS client: " + err.Error())
	}

	b := &BaiduBOSBackend{
		Client: client,
		Bucket: bucket,
		Prefix: cleanPrefix(prefix),
	}
	return b
}

// ListObjects lists all objects in Baidu Cloud BOS bucket, at prefix
func (b BaiduBOSBackend) ListObjects(prefix string) ([]Object, error) {
    var objects []Object

    prefix = pathutil.Join(b.Prefix, prefix)
    listObjectsArgs := &api.ListObjectsArgs{
        Prefix:  prefix,
        Marker:  "",
        MaxKeys: 1000,
    }
    for {
        lor, err := b.Client.ListObjects(b.Bucket, listObjectsArgs)
        if err != nil {
            return objects, err
        }

        for _, obj := range lor.Contents {
            path := removePrefixFromObjectPath(prefix, obj.Key)
            if objectPathIsInvalid(path) {
                continue
            }
            lastModified, err := time.Parse(time.RFC3339, obj.LastModified)
            if err != nil {
                continue
            }
            object := Object{
                Path:         path,
                Content:      []byte{},
                LastModified: lastModified,
            }
            objects = append(objects, object)
        }
        if !lor.IsTruncated {
            break
        }
        listObjectsArgs.Prefix = lor.Prefix
        listObjectsArgs.Marker = lor.NextMarker
    }

    return objects, nil
}

// GetObject retrieves an object from Baidu Cloud BOS bucket, at prefix
func (b BaiduBOSBackend) GetObject(path string) (Object, error) {
	var object Object
	object.Path = path
	var content []byte
	key := pathutil.Join(b.Prefix, path)
	bosObject, err := b.Client.BasicGetObject(b.Bucket, key)
	if err != nil {
		return object, err
	}
	body := bosObject.Body

	content, err = ioutil.ReadAll(body)
	body.Close()
	if err != nil {
		return object, err
	}
	object.Content = content

	meta, err := b.Client.GetObjectMeta(b.Bucket, key)
	if err != nil {
		return object, err
	}
	lastModified, err := time.Parse(time.RFC1123, meta.LastModified)
	object.LastModified = lastModified
	return object, nil
}

// PutObject uploads an object to Baidu Cloud BOS bucket, at prefix
func (b BaiduBOSBackend) PutObject(path string, content []byte) error {
	key := pathutil.Join(b.Prefix, path)
	var err error
	_, err = b.Client.PutObjectFromBytes(b.Bucket, key, content, nil)
	return err
}

// DeleteObject removes an object from Baidu Cloud BOS bucket, at prefix
func (b BaiduBOSBackend) DeleteObject(path string) error {
	key := pathutil.Join(b.Prefix, path)
	err := b.Client.DeleteObject(b.Bucket, key)
	return err
}
