package cache

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type StoreTestSuite struct {
	suite.Suite
	Stores map[string]Store
}

func (suite *StoreTestSuite) SetupSuite() {
	suite.Stores = make(map[string]Store)

	inMemoryStore := NewInMemoryStore()
	suite.Stores["InMemory"] = inMemoryStore

	if os.Getenv("TEST_REDIS") == "1" {
		redisStore := NewRedisStore(&RedisStoreOptions{
			Addr: "localhost:6379",
		})
		suite.Stores["Redis"] = redisStore
	}
}

func (suite *StoreTestSuite) TestAllStores() {
	for key, store := range suite.Stores {
		err := store.Set("x", []byte("1"))
		suite.Nil(err, fmt.Sprintf("able to create a new key using %s store", key))

		value, err := store.Get("x")
		suite.Nil(err, "able to get a key")
		suite.Equal([]byte("1"), value, fmt.Sprintf("able to get a key using %s store", key))

		err = store.Set("x", []byte("2"))
		suite.Nil(err, fmt.Sprintf("able to update an existing key using %s store", key))

		value, err = store.Get("x")
		suite.Nil(err, fmt.Sprintf("able to get a key after update using %s store", key))
		suite.Equal([]byte("2"), value, fmt.Sprintf("able to get a key after update using %s store", key))

		err = store.Delete("x")
		suite.Nil(err, fmt.Sprintf("able to delete a key using %s store", key))

		value, err = store.Get("x")
		suite.NotNil(err, fmt.Sprintf("error getting deleted key using %s store", key))
		suite.Nil(value, fmt.Sprintf("error getting deleted key using %s store", key))

		// in Redis, "A key is ignored if it does not exist"
		if key == "InMemory" {
			err = store.Delete("x")
			suite.NotNil(err, fmt.Sprintf("error deleting already-deleted key using %s store", key))
		}
	}
}

func TestStoreTestSuite(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}
