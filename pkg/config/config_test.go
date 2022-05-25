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
	"fmt"
	"io/ioutil"
	"os"
	pathutil "path"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/urfave/cli"
)

type ConfigTestSuite struct {
	suite.Suite
	TempDirectory  string
	TempConfigFile string
}

func (suite *ConfigTestSuite) SetupSuite() {
	timestamp := time.Now().Format("20060102150405")
	tempDirectory := fmt.Sprintf("../../.test/chartmuseum-config/%s", timestamp)
	os.MkdirAll(tempDirectory, os.ModePerm)
	suite.TempDirectory = tempDirectory

	tempConfigFile := pathutil.Join(tempDirectory, "chartmuseum.yaml")

	data := []byte(
		`
basicauth:
    user: "myuser"
    pass: "mypass"
`,
	)

	err := ioutil.WriteFile(tempConfigFile, data, 0644)
	suite.Nil(err, fmt.Sprintf("no error creating new config file %s", tempConfigFile))
	suite.TempConfigFile = tempConfigFile
}

func (suite *ConfigTestSuite) TearDownSuite() {
	err := os.RemoveAll(suite.TempDirectory)
	suite.Nil(err, "no error deleting temp directory")
}

func (suite *ConfigTestSuite) TestGetCLIFlagFromVarName() {
	suite.Equal("basic-auth-user", GetCLIFlagFromVarName("basicauth.user"))
	suite.Equal("", GetCLIFlagFromVarName("blah.blah.blah"))
}

func (suite *ConfigTestSuite) TestUpdateFromCLIContext() {
	var conf *Config
	var c *cli.Context
	var err error

	// no config file and empty context, everything should be set to default
	conf = NewConfig()
	suite.NotNil(conf)
	c = getNewContext()
	err = conf.UpdateFromCLIContext(c)
	suite.Nil(err)
	suite.Equal("", conf.GetString("charturl"))
	suite.Equal(8080, conf.GetInt("port"))
	suite.Equal(false, conf.GetBool("debug"))

	// no config file and populated context, everything should be set to what is in context
	conf = NewConfig()
	suite.NotNil(conf)
	c = getNewContext()
	c.Set("chart-url", "https://fakesite.com")
	c.Set("port", "8081")
	c.Set("debug", "true")
	conf.UpdateFromCLIContext(c)
	suite.Nil(err)
	suite.Equal("https://fakesite.com", conf.GetString("charturl"))
	suite.Equal(8081, conf.GetInt("port"))
	suite.Equal(true, conf.GetBool("debug"))

	// nonexistant config file, error
	conf = NewConfig()
	suite.NotNil(conf)
	c = getNewContext()
	c.Set("config", "thisisafakefile.yaml")
	err = conf.UpdateFromCLIContext(c)
	suite.NotNil(err)

	// config file with bad extension, error
	conf = NewConfig()
	suite.NotNil(conf)
	c = getNewContext()
	c.Set("config", "../../README.md")
	err = conf.UpdateFromCLIContext(c)
	suite.NotNil(err)

	// valid config file and empty context, everything should match config file
	conf = NewConfig()
	suite.NotNil(conf)
	c = getNewContext()
	c.Set("config", suite.TempConfigFile)
	err = conf.UpdateFromCLIContext(c)
	suite.Nil(err)
	suite.Equal("myuser", conf.GetString("basicauth.user"))
	suite.Equal("mypass", conf.GetString("basicauth.pass"))

	// valid config file and populated context, context vars should override config file
	conf = NewConfig()
	suite.NotNil(conf)
	c = getNewContext()
	c.Set("basic-auth-user", "otherdude")
	c.Set("config", suite.TempConfigFile)
	err = conf.UpdateFromCLIContext(c)
	suite.Nil(err)
	suite.Equal("otherdude", conf.GetString("basicauth.user"))
	suite.Equal("mypass", conf.GetString("basicauth.pass"))

	// Tests KeyValueFlag "generic" flag implementation
	conf = NewConfig()
	suite.NotNil(conf)
	c = getNewContext()
	c.Set("artifact-hub-repo-id", "foo=bar")
	err = conf.UpdateFromCLIContext(c)
	suite.Nil(err)
	suite.Equal(map[string]string{"foo": "bar"}, conf.GetStringMapString("artifact-hub-repo-id"))
}

func getNewContext() *cli.Context {
	var c *cli.Context
	app := cli.NewApp()
	app.Action = func(ctx *cli.Context) {
		c = ctx
	}
	app.Flags = CLIFlags
	app.Run([]string{"config.test"})
	return c
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
