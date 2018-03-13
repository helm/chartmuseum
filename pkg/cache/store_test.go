package cache

import (
	"fmt"
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
}

func (suite *StoreTestSuite) TestAllStores() {
	for key, store := range suite.Stores {
		err := store.Set("x", 1)
		suite.Nil(err, fmt.Sprintf("able to create a new key using %s store", key))

		value, err := store.Get("x")
		suite.Nil(err, "able to get a key")
		suite.Equal(1, value, fmt.Sprintf("able to get a key using %s store", key))

		err = store.Set("x", 2)
		suite.Nil(err, fmt.Sprintf("able to update an existing key using %s store", key))

		value, err = store.Get("x")
		suite.Nil(err, fmt.Sprintf("able to get a key after update using %s store", key))
		suite.Equal(2, value, fmt.Sprintf("able to get a key after update using %s store", key))

		err = store.Delete("x")
		suite.Nil(err, fmt.Sprintf("able to delete a key using %s store", key))

		value, err = store.Get("x")
		suite.NotNil(err, fmt.Sprintf("error getting deleted key using %s store", key))
		suite.Nil(value, fmt.Sprintf("error getting deleted key using %s store", key))

		err = store.Delete("x")
		suite.NotNil(err, fmt.Sprintf("error deleting already-deleted key using %s store", key))
	}
}

func TestStoreTestSuite(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}
