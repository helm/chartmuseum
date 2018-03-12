package cache

import (
	"errors"
	"fmt"
	"sync"
)

type (
	// InMemoryStore implements the Store interface, used for storing objects in-memory
	InMemoryStore struct {
		Keys    map[string]*InMemoryObject
		KeyLock *sync.Mutex
	}

	// InMemoryObject represents a single in-memory key value
	InMemoryObject struct {
		Contents  interface{}
		WriteLock *sync.Mutex
	}
)

// NewInMemoryObject creates a new, empty InMemoryStore
func NewInMemoryStore() *InMemoryStore {
	store := &InMemoryStore{
		Keys:    map[string]*InMemoryObject{},
		KeyLock: &sync.Mutex{},
	}
	return store
}

// Get returns an object at key
func (store *InMemoryStore) Get(key string) (interface{}, error) {
	if object, ok := store.Keys[key]; ok {
		return object.Contents, nil
	}
	return nil, errors.New(fmt.Sprintf("Could not find key \"%s\"", key))
}

// Set saves a new value for key
func (store *InMemoryStore) Set(key string, contents interface{}) error {
	if object, ok := store.Keys[key]; ok {
		object.WriteLock.Lock()
		object.Contents = contents
		object.WriteLock.Unlock()
	} else {
		store.KeyLock.Lock()
		store.Keys[key] = &InMemoryObject{
			Contents:  contents,
			WriteLock: &sync.Mutex{},
		}
		store.KeyLock.Unlock()
	}
	return nil
}

// Delete removes a key from the store
func (store *InMemoryStore) Delete(key string) error {
	if _, ok := store.Keys[key]; ok {
		store.KeyLock.Lock()
		delete(store.Keys, key)
		store.KeyLock.Unlock()
		return nil
	}
	return errors.New(fmt.Sprintf("Could not find key \"%s\"", key))
}
