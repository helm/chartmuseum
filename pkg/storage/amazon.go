package storage

import (
	"bytes"
	"io/ioutil"
	pathutil "path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// AmazonS3Backend is a storage backend for Amazon S3
type AmazonS3Backend struct {
	Bucket     string
	Client     *s3.S3
	Downloader *s3manager.Downloader
	Prefix     string
	Uploader   *s3manager.Uploader
	SSE        string
}

// NewAmazonS3Backend creates a new instance of AmazonS3Backend
func NewAmazonS3Backend(bucket string, prefix string, region string, endpoint string, sse string) *AmazonS3Backend {
	service := s3.New(session.New(), &aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(endpoint),
		DisableSSL:       aws.Bool(strings.HasPrefix(endpoint, "http://")),
		S3ForcePathStyle: aws.Bool(endpoint != ""),
	})
	b := &AmazonS3Backend{
		Bucket:     bucket,
		Client:     service,
		Downloader: s3manager.NewDownloaderWithClient(service),
		Prefix:     cleanPrefix(prefix),
		Uploader:   s3manager.NewUploaderWithClient(service),
		SSE:        sse,
	}
	return b
}

// ListObjects lists all objects in Amazon S3 bucket, at prefix
func (b AmazonS3Backend) ListObjects(prefix string) ([]Object, error) {
	var objects []Object
	prefix = pathutil.Join(b.Prefix, prefix)
	s3Input := &s3.ListObjectsInput{
		Bucket: aws.String(b.Bucket),
		Prefix: aws.String(prefix),
	}
	for {
		s3Result, err := b.Client.ListObjects(s3Input)
		if err != nil {
			return objects, err
		}
		for _, obj := range s3Result.Contents {
			path := removePrefixFromObjectPath(prefix, *obj.Key)
			if objectPathIsInvalid(path) {
				continue
			}
			object := Object{
				Path:         path,
				Content:      []byte{},
				LastModified: *obj.LastModified,
			}
			objects = append(objects, object)
		}
		if !*s3Result.IsTruncated {
			break
		}
		s3Input.Marker = s3Result.Contents[len(s3Result.Contents)-1].Key
	}
	return objects, nil
}

// GetObject retrieves an object from Amazon S3 bucket, at prefix
func (b AmazonS3Backend) GetObject(path string) (Object, error) {
	var object Object
	object.Path = path
	var content []byte
	s3Input := &s3.GetObjectInput{
		Bucket: aws.String(b.Bucket),
		Key:    aws.String(pathutil.Join(b.Prefix, path)),
	}
	s3Result, err := b.Client.GetObject(s3Input)
	if err != nil {
		return object, err
	}
	content, err = ioutil.ReadAll(s3Result.Body)
	if err != nil {
		return object, err
	}
	object.Content = content
	object.LastModified = *s3Result.LastModified
	return object, nil
}

// PutObject uploads an object to Amazon S3 bucket, at prefix
func (b AmazonS3Backend) PutObject(path string, content []byte) error {
	s3Input := &s3manager.UploadInput{
		Bucket: aws.String(b.Bucket),
		Key:    aws.String(pathutil.Join(b.Prefix, path)),
		Body:   bytes.NewBuffer(content),
	}

	if b.SSE != "" {
		s3Input.ServerSideEncryption = aws.String(b.SSE)
	}

	_, err := b.Uploader.Upload(s3Input)
	return err
}

// DeleteObject removes an object from Amazon S3 bucket, at prefix
func (b AmazonS3Backend) DeleteObject(path string) error {
	s3Input := &s3.DeleteObjectInput{
		Bucket: aws.String(b.Bucket),
		Key:    aws.String(pathutil.Join(b.Prefix, path)),
	}
	_, err := b.Client.DeleteObject(s3Input)
	return err
}
