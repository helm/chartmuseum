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

package multitenant

import (
	"errors"
	"sync"
	"testing"
	"time"

	cm_storage "github.com/chartmuseum/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "helm.sh/chartmuseum/pkg/repo"
)

// failingBackend holds every GetObject call until all of the expected
// callers have arrived, then fails them together.
type failingBackend struct {
	cm_storage.Backend
	arrived sync.WaitGroup
}

func (b *failingBackend) GetObject(path string) (cm_storage.Object, error) {
	b.arrived.Done()
	b.arrived.Wait()
	return cm_storage.Object{}, errors.New("storage unavailable")
}

func TestAddIndexObjectsAsyncReleasesWorkersOnError(t *testing.T) {
	const workers = 3

	logger, err := cm_logger.NewLogger(cm_logger.LoggerOptions{})
	require.NoError(t, err)

	backend := &failingBackend{Backend: cm_storage.NewLocalFilesystemBackend(t.TempDir())}
	backend.arrived.Add(workers)

	server := &MultiTenantServer{
		StorageBackend: backend,
		IndexLimit:     workers,
		Limiter:        make(chan struct{}, workers),
	}

	objects := make([]cm_storage.Object, workers)
	for i := range objects {
		objects[i] = cm_storage.Object{Path: "chart-" + string(rune('a'+i)) + "-0.1.0.tgz"}
	}

	index := &cm_repo.Index{}
	err = server.addIndexObjectsAsync(logger.ContextLoggingFn(&gin.Context{}), "", index, objects)
	require.Error(t, err)

	// The workers that did not report the error first must still finish,
	// otherwise they keep their Limiter slot and later index builds stall.
	require.Eventually(t, func() bool {
		return len(server.Limiter) == 0
	}, 5*time.Second, 10*time.Millisecond, "workers did not exit after the first error")
}
