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

package main

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"helm.sh/chartmuseum/pkg/chartmuseum"

	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	suite.Suite
	RedisMock        *miniredis.Miniredis
	LastCrashMessage string
}

func (suite *MainTestSuite) SetupSuite() {
	crash = func(v ...interface{}) {
		suite.LastCrashMessage = fmt.Sprint(v...)
		panic(v)
	}
	newServer = func(options chartmuseum.ServerOptions) (chartmuseum.Server, error) {
		return nil, errors.New("graceful crash")
	}

	redisMock, err := miniredis.Run()
	suite.Nil(err, "able to create miniredis instance")
	suite.RedisMock = redisMock
}

func (suite *MainTestSuite) TearDownSuite() {
	suite.RedisMock.Close()
}

func (suite *MainTestSuite) TestMain() {
	os.Args = []string{"chartmuseum", "--config", "blahblahblah.yaml"}
	suite.Panics(main, "bad config")
	suite.Equal("config file not found: blahblahblah.yaml", suite.LastCrashMessage, "crashes with bad config")

	os.Args = []string{"chartmuseum"}
	suite.Panics(main, "no storage")
	suite.Equal("Missing required flags(s): --storage", suite.LastCrashMessage, "crashes with no storage")

	os.Args = []string{"chartmuseum", "--storage", "garage"}
	suite.Panics(main, "bad storage")
	suite.Equal("Unsupported storage backend: garage", suite.LastCrashMessage, "crashes with bad storage")

	os.Args = []string{"chartmuseum", "--storage", "local", "--storage-local-rootdir", "../../.chartstorage"}
	suite.Panics(main, "local storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with local backend")

	os.Args = []string{"chartmuseum", "--storage", "amazon", "--storage-amazon-bucket", "x", "--storage-amazon-region", "x"}
	suite.Panics(main, "amazon storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with amazon backend")

	os.Args = []string{"chartmuseum", "--storage", "amazon", "--storage-amazon-bucket", "x", "--storage-amazon-endpoint", "http://localhost:9000"}
	suite.Panics(main, "amazon storage, alt endpoint")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with amazon backend, alt endpoint")

	os.Args = []string{"chartmuseum", "--storage", "google", "--storage-google-bucket", "x"}
	suite.Panics(main, "google storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with google backend")

	os.Args = []string{"chartmuseum", "--storage", "microsoft", "--storage-microsoft-container", "x"}
	suite.Panics(main, "microsoft storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with microsoft backend")

	os.Args = []string{"chartmuseum", "--storage", "alibaba", "--storage-alibaba-bucket", "x", "--storage-alibaba-endpoint", "oss-cn-beijing.aliyuncs.com"}
	suite.Panics(main, "alibaba storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with alibaba backend")

	os.Args = []string{"chartmuseum", "--storage", "openstack", "--storage-openstack-container", "x", "--storage-openstack-region", "x"}
	suite.Panics(main, "openstack storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with openstack backend")

	os.Args = []string{"chartmuseum", "--storage", "oracle", "--storage-oracle-bucket", "x", "--storage-oracle-region", "x", "--storage-oracle-compartmentid", "x"}
	suite.Panics(main, "oracle storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with oracle backend")

	os.Args = []string{"chartmuseum", "--storage", "baidu", "--storage-baidu-bucket", "x", "--storage-baidu-endpoint", "bj.bcebos.com"}
	suite.Panics(main, "baidu storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with baidu backend")

	// Redis cache
	os.Args = []string{"chartmuseum", "--storage", "local", "--storage-local-rootdir", "../../.chartstorage", "--cache", "redis", "--cache-redis-addr", suite.RedisMock.Addr()}
	suite.Panics(main, "redis cache")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with redis cache")

	// Unsupported cache store
	os.Args = []string{"chartmuseum", "--storage", "local", "--storage-local-rootdir", "../../.chartstorage", "--cache", "wallet"}
	suite.Panics(main, "bad cache")
	suite.Equal("Unsupported cache store: wallet", suite.LastCrashMessage, "crashes with bad cache")

}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
