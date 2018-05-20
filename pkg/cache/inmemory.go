package cache

import (
	"errors"
	"fmt"

	"github.com/coocood/freecache"
)

type (
	// InMemoryStore implements the Store interface, used for storing objects in-memory
	InMemoryStore struct {
		Cache *freecache.Cache
	}
)

// NewInMemoryObject creates a new, empty InMemoryStore
func NewInMemoryStore(size int) *InMemoryStore {
	cache := freecache.NewCache(size)
	store := &InMemoryStore{cache}
	return store
}

// Get returns an object at key
func (store *InMemoryStore) Get(key string) ([]byte, error) {
	return store.Cache.Get([]byte(key))
}

// Set saves a new value for key
func (store *InMemoryStore) Set(key string, contents []byte) error {
	return store.Cache.Set([]byte(key), contents, 0)
}

// Delete removes a key from the store
func (store *InMemoryStore) Delete(key string) error {
	if affected := store.Cache.Del([]byte(key)); !affected {
		return errors.New(fmt.Sprintf("unable to find key \"%s\"in cache", key))
	}
	return nil
}
