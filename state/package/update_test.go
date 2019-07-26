package pkg

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type UpdateTestSuite struct {
	PkgTestSuite
}

func (suite *UpdateTestSuite) TestUpdate() {
	suite.runsCommand([]string{"update", "artifact"}, -1, "Package updated: artifact")
}

func (suite *UpdateTestSuite) TestUpdateVersion() {
	suite.runsCommand([]string{"update", "artifact@2.0"}, -1, "Package updated: artifact@2.0")
}

func (suite *UpdateTestSuite) TestUpdateInvalidVersion() {
	suite.runsCommand([]string{"update", "artifact@10.0"}, 1, "provided package does not exist")
}

func TestUpdateTestSuite(t *testing.T) {
	suite.Run(t, new(UpdateTestSuite))
}
