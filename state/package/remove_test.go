package pkg

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type RemoveTestSuite struct {
	PkgTestSuite
}

func (suite *RemoveTestSuite) TestRemove() {
	suite.runsCommand([]string{"remove", "artifact"}, -1, "Package removed: artifact")
}

func TestRemoveTestSuite(t *testing.T) {
	suite.Run(t, new(RemoveTestSuite))
}
