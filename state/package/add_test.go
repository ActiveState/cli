package pkg

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type AddTestSuite struct {
	PkgTestSuite
}

func (suite *AddTestSuite) TestAdd() {
	suite.runsCommand([]string{"add", "artifact"}, -1, "Package added: artifact")
}

func (suite *AddTestSuite) TestAddVersion() {
	suite.runsCommand([]string{"add", "artifact@2.0"}, -1, "Package added: artifact@2.0")
}

func (suite *AddTestSuite) TestAddInvalidVersion() {
	suite.runsCommand([]string{"add", "artifact@10.0"}, 1, "provided package does not exist")
}

func TestAddTestSuite(t *testing.T) {
	suite.Run(t, new(AddTestSuite))
}
