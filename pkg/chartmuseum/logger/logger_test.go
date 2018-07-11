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

package logger

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/gin-gonic/gin"
)

type LoggerTestSuite struct {
	suite.Suite
	Logger *Logger
	Context *gin.Context
}

func (suite *LoggerTestSuite) SetupSuite() {
	logger, err := NewLogger(LoggerOptions{
		Debug:   false,
		LogJSON: false,
	})
	suite.Nil(err, "No err creating Logger, json=false, debug=false")

	logger, err = NewLogger(LoggerOptions{
		Debug:   false,
		LogJSON: true,
	})
	suite.Nil(err, "No err creating Logger, json=false, debug=true")

	logger, err = NewLogger(LoggerOptions{
		Debug:   true,
		LogJSON: false,
	})
	suite.Nil(err, "No err creating Logger, json=true, debug=false")

	logger, err = NewLogger(LoggerOptions{
		Debug:   true,
		LogJSON: true,
	})
	suite.Nil(err, "No err creating Logger, json=true, debug=true")
	suite.Logger = logger

	context := &gin.Context{}
	context.Set("requestcount", "1")
	context.Set("requestid", "xyz")
	suite.Context = context
}

func (suite *LoggerTestSuite) TestLevelcMethods() {
	suite.Logger.Debugc(suite.Context, "Debugc test", "x", "y")
	suite.Logger.Infoc(suite.Context, "Infoc test", "x", "y")
	suite.Logger.Warnc(suite.Context, "Warnc test", "x", "y")
	suite.Logger.Errorc(suite.Context, "Errorc test", "x", "y")
}

func (suite *LoggerTestSuite) TestContextLoggingFn() {
	log := suite.Logger.ContextLoggingFn(suite.Context)
	log(DebugLevel, "ContextLoggingFn debug test", "x", "y")
	log(InfoLevel, "ContextLoggingFn info test", "x", "y")
	log(WarnLevel, "ContextLoggingFn warn test", "x", "y")
	log(ErrorLevel, "ContextLoggingFn error test", "x", "y")
}

func TestLoggerTestSuite(t *testing.T) {
	suite.Run(t, new(LoggerTestSuite))
}
