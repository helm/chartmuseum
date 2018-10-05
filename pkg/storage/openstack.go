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
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	pathutil "path"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	osObjects "github.com/gophercloud/gophercloud/openstack/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/pagination"
)

// ReauthRoundTripper satisfies the http.RoundTripper interface and is used to
// limit the number of consecutive re-auth attempts (infinite by default)
type ReauthRoundTripper struct {
	rt                http.RoundTripper
	numReauthAttempts int
}

// RoundTrip performs a round-trip HTTP request and logs relevant information about it.
func (rrt *ReauthRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := rrt.rt.RoundTrip(request)
	if response == nil {
		return nil, err
	}

	if response.StatusCode == http.StatusUnauthorized {
		if rrt.numReauthAttempts == 3 {
			return response, errors.New("tried to re-authenticate 3 times with no success")
		}
		rrt.numReauthAttempts++
	} else {
		rrt.numReauthAttempts = 0
	}

	return response, nil
}

// OpenstackOSBackend is a storage backend for Openstack Object Storage
type OpenstackOSBackend struct {
	Container string
	Prefix    string
	Region    string
	CACert    string
	Client    *gophercloud.ServiceClient
}

// NewOpenstackOSBackend creates a new instance of OpenstackOSBackend
func NewOpenstackOSBackend(container string, prefix string, region string, caCert string) *OpenstackOSBackend {
	authOptions, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		panic(fmt.Sprintf("Openstack (environment): %s", err))
	}
	authOptions.AllowReauth = true

	// Create a custom HTTP client to handle reauth retry and custom CACERT if needed
	roundTripper := ReauthRoundTripper{}
	if caCert != "" {
		caCert, err := ioutil.ReadFile(caCert)
		if err != nil {
			panic(fmt.Sprintf("Openstack (ca certificates): %s", err))
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			panic(fmt.Sprintf("Openstack (ca certificates): unable to read certificate bundle"))
		}

		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}
		roundTripper.rt = transport
	} else {
		roundTripper.rt = http.DefaultTransport
	}

	provider, err := openstack.NewClient(authOptions.IdentityEndpoint)
	if err != nil {
		panic(fmt.Sprintf("Openstack (client): %s", err))
	}

	provider.HTTPClient = http.Client{
		Transport: &roundTripper,
	}

	err = openstack.Authenticate(provider, authOptions)
	if err != nil {
		panic(fmt.Sprintf("Openstack (authenticate): %s", err))
	}

	client, err := openstack.NewObjectStorageV1(provider, gophercloud.EndpointOpts{
		Region: region,
	})
	if err != nil {
		panic(fmt.Sprintf("Openstack (object storage): %s", err))
	}

	b := &OpenstackOSBackend{
		Container: container,
		Prefix:    prefix,
		Region:    region,
		Client:    client,
	}

	return b
}

// ListObjects lists all objects in an Openstack container, at prefix
func (b OpenstackOSBackend) ListObjects(prefix string) ([]Object, error) {
	var objects []Object

	prefix = pathutil.Join(b.Prefix, prefix)
	opts := &osObjects.ListOpts{
		Full:   true,
		Prefix: prefix,
	}

	pager := osObjects.List(b.Client, b.Container, opts)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		objectList, err := osObjects.ExtractInfo(page)
		if err != nil {
			return false, err
		}

		for _, openStackObject := range objectList {
			path := removePrefixFromObjectPath(prefix, openStackObject.Name)
			if objectPathIsInvalid(path) {
				continue
			}
			object := Object{
				Path:         path,
				Content:      []byte{},
				LastModified: openStackObject.LastModified,
			}
			objects = append(objects, object)
		}
		return true, nil
	})

	return objects, err
}

// GetObject retrieves an object from an Openstack container, at prefix
func (b OpenstackOSBackend) GetObject(path string) (Object, error) {
	var object Object
	object.Path = path

	result := osObjects.Download(b.Client, b.Container, pathutil.Join(b.Prefix, path), nil)
	headers, err := result.Extract()
	if err != nil {
		return object, err
	}
	object.LastModified = headers.LastModified

	content, err := result.ExtractContent()
	if err != nil {
		return object, err
	}
	object.Content = content
	return object, nil
}

// PutObject uploads an object to Openstack container, at prefix
func (b OpenstackOSBackend) PutObject(path string, content []byte) error {
	reader := bytes.NewReader(content)
	createOpts := osObjects.CreateOpts{
		Content: reader,
	}
	_, err := osObjects.Create(b.Client, b.Container, pathutil.Join(b.Prefix, path), createOpts).Extract()
	return err
}

// DeleteObject removes an object from an Openstack container, at prefix
func (b OpenstackOSBackend) DeleteObject(path string) error {
	_, err := osObjects.Delete(b.Client, b.Container, pathutil.Join(b.Prefix, path), nil).Extract()
	return err
}
