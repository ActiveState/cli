package pkg

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
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

func (suite *AddTestSuite) TestAdd_CommitError() {
	httpmock.RegisterWithCode("PUT", "/vcs/branch/00010001-0001-0001-0001-000100010001", 404)
	suite.runsCommand([]string{"add", "artifact"}, 1, "Failed to add package")
}

func TestAddTestSuite(t *testing.T) {
	suite.Run(t, new(AddTestSuite))
}
