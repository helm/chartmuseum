package router

import (
	"testing"

	"github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	"github.com/stretchr/testify/suite"
)

type RouterGroupsTestSuite struct {
	suite.Suite
}

func (suite *RouterGroupsTestSuite) TestNewRouterGroups() {
	log, err := logger.NewLogger(logger.LoggerOptions{
		Debug:   true,
		LogJSON: true,
	})
	suite.Nil(err, "no error creating logger")

	router := NewRouter(RouterOptions{
		Logger: log,
	})

	// Open
	rgs := NewRouterGroups(RouterGroupsOptions{
		Logger: log,
		Router: router,
	})
	suite.NotNil(rgs)

	// Basic Auth
	rgs = NewRouterGroups(RouterGroupsOptions{
		Logger:       log,
		Router:       router,
		Username:     "josh",
		Password:     "dolphin",
		AnonymousGet: true,
	})
	suite.NotNil(rgs)

	// Basic Auth (protected GET)
	rgs = NewRouterGroups(RouterGroupsOptions{
		Logger:   log,
		Router:   router,
		Username: "josh",
		Password: "dolphin",
	})
	suite.NotNil(rgs)
}

func TestRouterGroupsTestSuite(t *testing.T) {
	suite.Run(t, new(RouterGroupsTestSuite))
}
