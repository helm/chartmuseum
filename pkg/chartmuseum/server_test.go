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

package chartmuseum

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/chartmuseum/storage"
	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"

	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite
	Backend       storage.Backend
	TempDirectory string
}

func (suite *ServerTestSuite) SetupSuite() {
	timestamp := time.Now().Format("20060102150405")
	brokenTempDirectory := fmt.Sprintf("../../.test/chartmuseum-server/%s", timestamp)
	suite.Backend = storage.Backend(storage.NewLocalFilesystemBackend(brokenTempDirectory))
}

func (suite *ServerTestSuite) TearDownSuite() {
	err := os.RemoveAll(suite.TempDirectory)
	suite.Nil(err, "no error deleting temp directory for local storage")
}

func (suite *ServerTestSuite) TestNewServer() {
	log, err := cm_logger.NewLogger(cm_logger.LoggerOptions{
		Debug:   true,
		LogJSON: true,
	})
	suite.Nil(err, "no error creating logger")
	serverOptions := ServerOptions{
		StorageBackend: suite.Backend,
		Logger:         log,
	}

	multiTenantServer, err := NewServer(serverOptions)
	suite.NotNil(multiTenantServer)
	suite.Nil(err)
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
