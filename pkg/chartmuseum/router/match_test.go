package router

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"net/http/httptest"
)

type MatchTestSuite struct {
	suite.Suite
}

func (suite *MatchTestSuite) TestMatch() {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	handlers := []gin.HandlerFunc{}

	for i := 0; i <= 9; i++ {
		{
			j := i
			handlers = append(handlers, func(c *gin.Context) {
				c.Set("index", j)
			})
		}
	}

	routes := []*Route{
		{"GET", "/", handlers[0]},
		{"GET", "/health", handlers[1]},
		{"GET", "/:repo/index.yaml", handlers[2]},
		{"GET", "/:repo/charts/:filename", handlers[3]},
		{"GET", "/api/:repo/charts", handlers[4]},
		{"GET", "/api/:repo/charts/:name", handlers[5]},
		{"GET", "/api/:repo/charts/:name/:version", handlers[6]},
		{"POST", "/api/:repo/charts", handlers[7]},
		{"POST", "/api/:repo/prov", handlers[8]},
		{"DELETE", "/api/:repo/charts/:name/:version", handlers[9]},
	}

	// /
	handle, params := match(routes, "/", "", 0)
	suite.NotNil(handle)
	suite.Nil(params)
	if handle != nil { handle(c) }
	val, exists := c.Get("index")
	suite.True(exists)
	suite.Equal(0, val)


	// /health
	handle, params = match(routes, "/health", "", 0)
	suite.NotNil(handle)
	suite.Nil(params)
	if handle != nil { handle(c) }
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(1, val)

	// /index.yaml
	handle, params = match(routes, "/index.yaml", "", 0)
	suite.NotNil(handle)
	if handle != nil { handle(c) }
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(2, val)
	suite.Equal([]gin.Param{{"repo", ""}}, params)

	// /myrepo/index.yaml
	handle, params = match(routes, "/myrepo/index.yaml", "", 1)
	suite.NotNil(handle)
	if handle != nil { handle(c) }
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(2, val)
	suite.Equal([]gin.Param{{"repo", "myrepo"}}, params)

	// /myorg/myrepo/index.yaml
	handle, params = match(routes, "/myorg/myrepo/index.yaml", "", 2)
	suite.NotNil(handle)
	if handle != nil { handle(c) }
	val, exists = c.Get("index")
	suite.True(exists)
	suite.Equal(2, val)
	suite.Equal([]gin.Param{{"repo", "myorg/myrepo"}}, params)
}

func TestMatchTestSuite(t *testing.T) {
	suite.Run(t, new(MatchTestSuite))
}
