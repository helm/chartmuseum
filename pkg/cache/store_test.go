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
	"fmt"
	"testing"

	"github.com/alicebob/miniredis"
	"github.com/stretchr/testify/suite"
)

type StoreTestSuite struct {
	suite.Suite
	RedisMock *miniredis.Miniredis
	Stores    map[string]Store
}

func (suite *StoreTestSuite) SetupSuite() {
	suite.Stores = make(map[string]Store)

	redisMock, err := miniredis.Run()
	suite.Nil(err, "able to create miniredis instance")
	suite.RedisMock = redisMock
	suite.Stores["Redis"] = NewRedisStore(redisMock.Addr(), "", 0)
}

func (suite *StoreTestSuite) TearDownSuite() {
	suite.RedisMock.Close()
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
		suite.Equal([]byte{}, value, fmt.Sprintf("error getting deleted key using %s store", key))

		// in Redis, "A key is ignored if it does not exist"
		if key != "Redis" {
			err = store.Delete("x")
			suite.NotNil(err, fmt.Sprintf("error deleting already-deleted key using %s store", key))
		}
	}
}

func TestStoreTestSuite(t *testing.T) {
	suite.Run(t, new(StoreTestSuite))
}
