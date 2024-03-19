package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ResetIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ResetIntegrationTestSuite) TestRevert() {
	suite.OnlyRunForTags(tagsuite.Reset)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Branches#35af7414-b44b-4fd7-aa93-2ecad337ed2b", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "requests")
	cp.Expect("Package added")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("history")
	cp.Expect("requests")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("reset")
	cp.Expect("Your project will be reset to 35af7414-b44b-4fd7-aa93-2ecad337ed2b")
	cp.Expect("Are you sure")
	cp.Expect("(y/N)")
	cp.SendLine("y")
	cp.Expect("Successfully reset to commit: 35af7414-b44b-4fd7-aa93-2ecad337ed2b")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("history")
	cp.ExpectExitCode(0)
	suite.Assert().NotContains(cp.Snapshot(), "requests")

	cp = ts.Spawn("reset")
	cp.Expect("You are already on the latest commit")
	cp.ExpectNotExitCode(0)

	cp = ts.Spawn("reset", "00000000-0000-0000-0000-000000000000")
	cp.Expect("The given commit ID does not exist")
	cp.ExpectNotExitCode(0)
}

func (suite *ResetIntegrationTestSuite) TestRevertToBranch() {
	suite.OnlyRunForTags(tagsuite.Reset)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Branches#46c83477-d580-43e2-a0c6-f5d3677517f1", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("reset", "main", "-n")
	cp.Expect("Successfully reset to commit: 35af7414-b44b-4fd7-aa93-2ecad337ed2b")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("reset", "main")
	cp.Expect("Your project is already at the given commit ID")
	cp.ExpectNotExitCode(0)

	cp = ts.Spawn("reset", "does-not-exist")
	cp.Expect("This project has no branch with label matching 'does-not-exist'")
	cp.ExpectNotExitCode(0)
}

func (suite *ResetIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Revert, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Branches#46c83477-d580-43e2-a0c6-f5d3677517f1", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("reset", "main", "-o", "json")
	cp.Expect(`{"commitID":"35af7414-b44b-4fd7-aa93-2ecad337ed2b"}`)
	cp.ExpectExitCode(0)
}

func TestResetIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ResetIntegrationTestSuite))
}
