package integration

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal-as/testhelpers/e2e"
	"github.com/ActiveState/cli/internal-as/testhelpers/tagsuite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type BranchIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *BranchIntegrationTestSuite) TestBranch_List() {
	suite.OnlyRunForTags(tagsuite.Branches)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts, "ActiveState-CLI", "Branches")

	cp := ts.Spawn("branch")
	expected := `main (Current)
 ├─ firstbranch
 │  └─ firstbranchchild
 │     └─ childoffirstbranchchild
 ├─ secondbranch
 └─ thirdbranch
`
	cp.ExpectLongString(expected)
	cp.ExpectExitCode(0)
}

func (suite *BranchIntegrationTestSuite) TestBranch_Add() {
	suite.OnlyRunForTags(tagsuite.Branches)
	suite.T().Skip("Skip test as state branch add functionality is currently disabled")
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts, e2e.PersistentUsername, "Branch")

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("pull")
	cp.ExpectLongString("Your project in the activestate.yaml has been updated")
	cp.ExpectExitCode(0)

	branchName, err := uuid.NewRandom()
	suite.Require().NoError(err)

	cp = ts.Spawn("branch", "add", branchName.String())
	cp.ExpectExitCode(0)

	cp = ts.Spawn("branch")
	cp.Expect(branchName.String())
	cp.ExpectExitCode(0)
}

func (suite *BranchIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session, username, project string) {
	asyData := fmt.Sprintf(`project: "https://platform.activestate.com/%s/%s"`, username, project)
	ts.PrepareActiveStateYAML(asyData)
}

func TestBranchIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BranchIntegrationTestSuite))
}
