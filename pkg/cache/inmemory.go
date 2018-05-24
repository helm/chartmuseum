package cache

import (
	"github.com/allegro/bigcache"
)

type (
	// InMemoryStore implements the Store interface, used for storing objects in-memory
	InMemoryStore struct {
		Cache *bigcache.BigCache
	}
)

// NewInMemoryObject creates a new, empty InMemoryStore
func NewInMemoryStore(maxEntries int, maxEntrySizeMb int, maxCacheSizeMb int) *InMemoryStore {
	cache, err := bigcache.NewBigCache(bigcache.Config{
		Shards:             maxEntries,
		MaxEntriesInWindow: maxEntries,
		MaxEntrySize:       maxEntrySizeMb * 1024 * 1024,
		HardMaxCacheSize:   maxCacheSizeMb,
	})
	if err != nil {
		panic(err)
	}
	store := &InMemoryStore{cache}
	return store
}

// Get returns an object at key
func (store *InMemoryStore) Get(key string) ([]byte, error) {
	return store.Cache.Get(key)
}

// Set saves a new value for key
func (store *InMemoryStore) Set(key string, contents []byte) error {
	return store.Cache.Set(key, contents)
}

// Delete removes a key from the store
func (store *InMemoryStore) Delete(key string) error {
	return store.Cache.Delete(key)
}
