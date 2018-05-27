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
func NewRedisStore(addr string, password string, db int) *RedisStore {
	store := &RedisStore{}
	redisClientOptions := &redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	}
	store.Client = redis.NewClient(redisClientOptions)
	return store
}

// Get returns an object at key
func (store *RedisStore) Get(key string) ([]byte, error) {
	content, err := store.Client.Get(key).Bytes()
	return content, err
}

// Set saves a new value for key
func (store *RedisStore) Set(key string, contents []byte) error {
	err := store.Client.Set(key, contents, 0).Err()
	return err
}

// Delete removes a key from the store
func (store *RedisStore) Delete(key string) error {
	err := store.Client.Del(key).Err()
	return err
}
