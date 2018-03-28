package router

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type MatchTestSuite struct {
	suite.Suite
}

func TestMatchTestSuite(t *testing.T) {
	suite.Run(t, new(MatchTestSuite))
}
