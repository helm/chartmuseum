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
