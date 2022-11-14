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

package cache

import (
	"sync"

	"helm.sh/chartmuseum/pkg/chartmuseum/events"
	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
	cm_repo "helm.sh/chartmuseum/pkg/repo"
)

type (
	CacheEntry struct {
		// cryptic JSON field names to minimize size saved in cache
		RepoName  string         `json:"a"`
		RepoIndex *cm_repo.Index `json:"b"`
		RepoLock  *sync.RWMutex
	}

	// Store is a generic interface for cache stores
	Store interface {
		Get(key string) ([]byte, error)
		Set(key string, contents []byte) error
		Delete(key string) error
		UpdateEntryFromEvent(key string, log cm_logger.LoggingFn, event events.Event, update func(log cm_logger.LoggingFn, entry *CacheEntry, event events.Event) error) error
	}
)
