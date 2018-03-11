package main

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum"

	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	suite.Suite
	LastCrashMessage string
}

func (suite *MainTestSuite) SetupSuite() {
	crash = func(v ...interface{}) {
		suite.LastCrashMessage = fmt.Sprint(v...)
		panic(v)
	}
	newServer = func(options chartmuseum.ServerOptions) (chartmuseum.Server, error) {
		return nil, errors.New("graceful crash")
	}
}

func (suite *MainTestSuite) TestMain() {
	os.Args = []string{"chartmuseum"}
	suite.Panics(main, "no storage")
	suite.Equal("Missing required flags(s): --storage", suite.LastCrashMessage, "crashes with no storage")

	os.Args = []string{"chartmuseum", "--storage", "garage"}
	suite.Panics(main, "bad storage")
	suite.Equal("Unsupported storage backend: garage", suite.LastCrashMessage, "crashes with bad storage")

	os.Args = []string{"chartmuseum", "--storage", "local", "--storage-local-rootdir", "../../.chartstorage"}
	suite.Panics(main, "local storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with local backend")

	os.Args = []string{"chartmuseum", "--storage", "amazon", "--storage-amazon-bucket", "x", "--storage-amazon-region", "x"}
	suite.Panics(main, "amazon storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with amazon backend")

	os.Args = []string{"chartmuseum", "--storage", "amazon", "--storage-amazon-bucket", "x", "--storage-amazon-endpoint", "http://localhost:9000"}
	suite.Panics(main, "amazon storage, alt endpoint")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with amazon backend, alt endpoint")

	os.Args = []string{"chartmuseum", "--storage", "google", "--storage-google-bucket", "x"}
	suite.Panics(main, "google storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with google backend")

	os.Args = []string{"chartmuseum", "--storage", "microsoft", "--storage-microsoft-container", "x"}
	suite.Panics(main, "microsoft storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with microsoft backend")

	os.Args = []string{"chartmuseum", "--storage", "alibaba", "--storage-alibaba-bucket", "x", "--storage-alibaba-endpoint", "oss-cn-beijing.aliyuncs.com"}
	suite.Panics(main, "alibaba storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with alibaba backend")
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
