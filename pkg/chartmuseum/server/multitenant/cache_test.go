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
	"fmt"
	"os"
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

func newTestObjects(n int) []cm_storage.Object {
	objects := make([]cm_storage.Object, n)
	for i := range objects {
		objects[i] = cm_storage.Object{Path: fmt.Sprintf("chart-%d-0.1.0.tgz", i)}
	}
	return objects
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

	// One deadline covers the rebuild itself and the wait for the slots,
	// so a hang fails this test instead of running into the package timeout.
	deadline := time.After(5 * time.Second)

	done := make(chan error, 1)
	go func() {
		done <- server.addIndexObjectsAsync(logger.ContextLoggingFn(&gin.Context{}), "", &cm_repo.Index{}, newTestObjects(workers))
	}()

	select {
	case err = <-done:
		require.EqualError(t, err, "storage unavailable")
	case <-deadline:
		t.Fatal("rebuild did not return after a failed chart load")
	}

	// The workers that did not report the error first must still finish,
	// otherwise they keep their Limiter slot and later index builds stall.
	for len(server.Limiter) != 0 {
		select {
		case <-deadline:
			t.Fatal("workers did not exit after the first error")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// countingBackend records the highest number of GetObject calls in flight
// so tests can assert it never exceeds the index limit.
type countingBackend struct {
	cm_storage.Backend
	mu       sync.Mutex
	inFlight int
	peak     int
	total    int
	content  []byte
}

func newCountingBackend(t *testing.T) *countingBackend {
	content, err := os.ReadFile("../../../../testdata/charts/mychart/mychart-0.1.0.tgz")
	require.NoError(t, err)
	return &countingBackend{
		Backend: cm_storage.NewLocalFilesystemBackend(t.TempDir()),
		content: content,
	}
}

func (b *countingBackend) fetched() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.total
}

func (b *countingBackend) GetObject(path string) (cm_storage.Object, error) {
	b.mu.Lock()
	b.inFlight++
	b.total++
	if b.inFlight > b.peak {
		b.peak = b.inFlight
	}
	b.mu.Unlock()

	time.Sleep(2 * time.Millisecond)

	b.mu.Lock()
	b.inFlight--
	b.mu.Unlock()
	return cm_storage.Object{Path: path, Content: b.content}, nil
}

func TestAddIndexObjectsAsyncRespectsIndexLimit(t *testing.T) {
	logger, err := cm_logger.NewLogger(cm_logger.LoggerOptions{})
	require.NoError(t, err)

	for _, limit := range []int{1, 2} {
		backend := newCountingBackend(t)
		server := &MultiTenantServer{
			StorageBackend: backend,
			IndexLimit:     limit,
			Limiter:        make(chan struct{}, limit),
		}

		done := make(chan error, 1)
		go func() {
			done <- server.addIndexObjectsAsync(logger.ContextLoggingFn(&gin.Context{}), "", cm_repo.NewIndex("", "", &cm_repo.ServerInfo{}, false), newTestObjects(32))
		}()
		select {
		case err = <-done:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("rebuild did not finish")
		}

		require.Equal(t, 32, backend.total)
		require.LessOrEqual(t, backend.peak, limit)
	}
}

// hookedLog builds a LoggingFn that only calls hook with each message. The
// logger's level type is unexported, so it is inferred from a sample level.
func hookedLog[L ~string](_ L, hook func(msg string)) func(L, string, ...interface{}) {
	return func(_ L, msg string, _ ...interface{}) { hook(msg) }
}

// With index processing paused, workers must not run ahead of the reader:
// a worker keeps its Limiter slot until its result has been received.
func TestAddIndexObjectsAsyncDoesNotRunAheadOfSlowIndexing(t *testing.T) {
	const objects = 16

	backend := newCountingBackend(t)
	server := &MultiTenantServer{
		StorageBackend: backend,
		IndexLimit:     1,
		Limiter:        make(chan struct{}, 1),
	}

	// Block the reader on its first result by stalling the log call that
	// precedes adding the chart to the index.
	paused := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	log := hookedLog(cm_logger.DebugLevel, func(msg string) {
		if msg == "Adding chart to index" {
			once.Do(func() { close(paused) })
			<-release
		}
	})

	deadline := time.After(10 * time.Second)

	done := make(chan error, 1)
	go func() {
		done <- server.addIndexObjectsAsync(log, "", cm_repo.NewIndex("", "", &cm_repo.ServerInfo{}, false), newTestObjects(objects))
	}()

	select {
	case <-paused:
	case <-deadline:
		t.Fatal("reader never received a result")
	}

	// Give any runaway workers time to start their fetches. The reader holds
	// one result and one worker holds the slot waiting to send the next.
	time.Sleep(200 * time.Millisecond)
	require.LessOrEqual(t, backend.fetched(), 2)

	close(release)
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-deadline:
		t.Fatal("rebuild did not finish after the reader resumed")
	}
	require.Equal(t, objects, backend.fetched())
}
