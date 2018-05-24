package chartmuseum

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kubernetes-helm/chartmuseum/pkg/storage"

	"github.com/kubernetes-helm/chartmuseum/pkg/cache"
	"github.com/stretchr/testify/suite"
)

type ServerTestSuite struct {
	suite.Suite
	Backend       storage.Backend
	Store         cache.Store
	TempDirectory string
}

func (suite *ServerTestSuite) SetupSuite() {
	timestamp := time.Now().Format("20060102150405")
	brokenTempDirectory := fmt.Sprintf("../../.test/chartmuseum-server/%s", timestamp)
	suite.Backend = storage.Backend(storage.NewLocalFilesystemBackend(brokenTempDirectory))
	suite.Store = cache.Store(cache.NewInMemoryStore(1, 1, 1))
}

func (suite *ServerTestSuite) TearDownSuite() {
	err := os.RemoveAll(suite.TempDirectory)
	suite.Nil(err, "no error deleting temp directory for local storage")
}

func (suite *ServerTestSuite) TestNewServer() {
	serverOptions := ServerOptions{
		StorageBackend: suite.Backend,
		CacheStore: suite.Store,
	}

	multiTenantServer, err := NewServer(serverOptions)
	suite.NotNil(multiTenantServer)
	suite.Nil(err)
}

func TestServerTestSuite(t *testing.T) {
	suite.Run(t, new(ServerTestSuite))
}
