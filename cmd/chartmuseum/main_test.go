package main

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/chartmuseum/chartmuseum/pkg/chartmuseum"
	"github.com/chartmuseum/chartmuseum/pkg/repo"

	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	suite.Suite
	LastPrinted      string
	LastExitCode     int
	LastCrashMessage string
}

func (suite *MainTestSuite) SetupSuite() {
	crash = func(v ...interface{}) {
		suite.LastCrashMessage = fmt.Sprint(v...)
		panic(v)
	}
	echo = func(v ...interface{}) (int, error) {
		suite.LastPrinted = fmt.Sprint(v...)
		return 0, nil
	}
	exit = func(code int) {
		suite.LastExitCode = code
		suite.LastCrashMessage = fmt.Sprintf("exited %d", code)
		panic(fmt.Sprintf("exited %d", code))
	}
	newServer = func(options chartmuseum.ServerOptions) (*chartmuseum.Server, error) {
		return &chartmuseum.Server{}, errors.New("graceful crash")
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

	os.Args = []string{"chartmuseum", "--storage", "google", "--storage-google-bucket", "x"}
	suite.Panics(main, "google storage")
	suite.Equal("graceful crash", suite.LastCrashMessage, "no error with google backend")

	// test the --gen-index option
	newServer = func(options chartmuseum.ServerOptions) (*chartmuseum.Server, error) {
		s := &chartmuseum.Server{}
		s.RepositoryIndex = repo.NewIndex("")
		s.RepositoryIndex.Regenerate()
		return s, nil
	}
	os.Args = []string{"chartmuseum", "--gen-index", "--storage", "local", "--storage-local-rootdir", "../../.chartstorage"}
	suite.Panics(main, "exited 0")
	suite.Equal("exited 0", suite.LastCrashMessage, "no error with --gen-index")
	suite.Equal(0, suite.LastExitCode, "--gen-index flag exits 0")
	suite.Contains(suite.LastPrinted, "apiVersion:", "--gen-index prints yaml")
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}
