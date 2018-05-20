package cache

import (
	"github.com/go-redis/redis"
)

type (
	// RedisStore implements the Store interface, used for storing objects in-memory
	RedisStore struct {
		Client *redis.Client
	}
)

// NewRedisStore creates a new RedisStore
func NewRedisStore(addr string) *RedisStore {
	store := &RedisStore{}
	redisClientOptions := &redis.Options{
		Addr: addr,
	}
	store.Client = redis.NewClient(redisClientOptions)
	return store
}

// Get returns an object at key
func (store *RedisStore) Get(key string) ([]byte, error) {
	return store.Client.Get(key).Bytes()
}

// Set saves a new value for key
func (store *RedisStore) Set(key string, contents []byte) error {
	return store.Client.Set(key, contents, 0).Err()
}

// Delete removes a key from the store
func (store *RedisStore) Delete(key string) error {
	return store.Client.Del(key).Err()
}
