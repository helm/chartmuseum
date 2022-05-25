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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/urfave/cli"

	cm_logger "helm.sh/chartmuseum/pkg/chartmuseum/logger"
)

type (
	// Config is a complete set of app configuration
	Config struct {
		*viper.Viper
	}
)

// NewConfig create a new Config instance
func NewConfig() *Config {
	conf := &Config{
		Viper: viper.New(),
	}
	conf.SetConfigType("yaml")
	conf.setDefaults()
	return conf
}

// GetCLIFlagFromVarName returns the name of the CLI flag associated with a config var
func GetCLIFlagFromVarName(name string) string {
	var val string
	if configVar, ok := configVars[name]; ok {
		if flag := configVar.CLIFlag; flag != nil {
			val = flag.GetName()
		}
	}
	return val
}

// UpdateFromCLIContext updates a config based on flags set in CLI context
func (conf *Config) UpdateFromCLIContext(c *cli.Context) error {
	err := conf.readConfigFileFromCLIContext(c)
	if err != nil {
		return err
	}

	for key, configVar := range configVars {
		if flag := configVar.CLIFlag; flag != nil {
			if name := flag.GetName(); c.IsSet(name) {
				switch configVar.Type {
				case stringType:
					conf.Set(key, c.String(name))
				case intType:
					conf.Set(key, c.Int(name))
				case boolType:
					conf.Set(key, c.Bool(name))
				case durationType:
					conf.Set(key, c.Duration(name))
				case keyValueType:
					keyValue, ok := c.Generic(name).(*KeyValueFlag)
					if !ok {
						return fmt.Errorf("failed to get flag value: %s", flag.GetName())
					}
					conf.Set(key, keyValue.m)
				}
			}
		}
	}

	return nil
}

func (conf *Config) ShowDeprecationWarnings(c *cli.Context, logger *cm_logger.Logger) {
	deprecatedOptions := []string{}
	deprecationCheck(c, configVars, &deprecatedOptions)
	for _, name := range deprecatedOptions {
		logger.Warnf("The configuration option %s has been deprecated", name)
	}
}

func (conf *Config) readConfigFileFromCLIContext(c *cli.Context) error {
	if confFilePath := c.String("config"); confFilePath != "" {
		if _, err := os.Stat(confFilePath); os.IsNotExist(err) {
			return errors.New(fmt.Sprintf("config file \"%s\" does not exist", confFilePath))
		}

		ext := filepath.Ext(confFilePath)
		if ext != ".yaml" && ext != ".yml" && ext != "" {
			return errors.New("config file must have .yaml/.yml extension (or no extension)")
		}

		base := strings.TrimSuffix(filepath.Base(confFilePath), ext)
		dir := filepath.Dir(confFilePath)
		conf.SetConfigName(base)
		conf.AddConfigPath(dir)
		return conf.ReadInConfig()
	}

	return nil
}

func (conf *Config) setDefaults() {
	for key, configVar := range configVars {
		conf.SetDefault(key, configVar.Default)
	}
}

func deprecationCheck(c *cli.Context, configVars map[string]configVar, deprecatedOptions *[]string) {
	for _, configVar := range configVars {
		if flag := configVar.CLIFlag; flag != nil {
			if name := flag.GetName(); c.IsSet(name) {
				if configVar.Deprecated {
					*deprecatedOptions = append(*deprecatedOptions, name)
				}
			}
		}
	}
}
