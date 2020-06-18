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

package config

import (
	// "strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// func TestConfigVarNames(t *testing.T) {
// 	for k, v := range configVars {
// 		// TODO cache.store can't match schema
// 		// TODO depthdynamic can't match schema
// 		// TODO storage.backend can't match schema
// 		if k == "cache.store" || k == "depthdynamic" || k == "storage.backend" {
// 			continue
// 		}
// 		should := strings.ReplaceAll(v.CLIFlag.GetName(), "-", ".")
// 		assert.Equal(t, should, k, "configVars key should be cli flag, dahes replaced with dots")
// 	}
// }

func TestCompatConfigVars(t *testing.T) {
	for alias, key := range aliasConfigVars {
		_, ok := configVars[key]
		assert.True(t, ok, "alias \"%s\" has bad reference: \"%s\"", alias, key)
	}
}
