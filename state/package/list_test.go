package pkg

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ListTestSuite struct {
	PkgTestSuite
}

func (suite *ListTestSuite) TestList() {
	suite.runsCommand(nil, -1, "Name")
}

func (suite *ListTestSuite) TestListByCommit() {
	commitID := "00090009-0009-0009-0009-000900090009"
	suite.runsCommand([]string{"--commit", commitID}, -1, "No data")

	suite.runsCommand([]string{"--commit", "latest"}, -1, "Name")
}

func TestListTestSuite(t *testing.T) {
	suite.Run(t, new(ListTestSuite))
}
