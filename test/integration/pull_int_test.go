package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type PullIntegrationTestSuite struct {
	suite.Suite
}

func TestPullIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PullIntegrationTestSuite))
}
