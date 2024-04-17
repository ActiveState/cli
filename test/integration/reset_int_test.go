package integration

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ResetIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ResetIntegrationTestSuite) TestReset() {
	suite.OnlyRunForTags(tagsuite.Reset)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Reset#3a2d095d-efd6-4be0-b824-21de94fc4ad6", ".")
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
	cp.Expect("Your project will be reset to 3a2d095d-efd6-4be0-b824-21de94fc4ad6")
	cp.Expect("Are you sure")
	cp.Expect("(y/N)")
	cp.SendLine("y")
	cp.Expect("Successfully reset to commit: 3a2d095d-efd6-4be0-b824-21de94fc4ad6")
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

	cp = ts.Spawn("reset", "does-not-exist")
	cp.Expect("This project has no branch with label matching 'does-not-exist'")
	cp.ExpectNotExitCode(0)
}

func (suite *ResetIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Reset, tagsuite.JSON)
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

func (suite *ResetIntegrationTestSuite) TestRevertInvalidURL() {
	suite.OnlyRunForTags(tagsuite.Reset)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	commitID := "3a2d095d-efd6-4be0-b824-21de94fc4ad6"

	cp := ts.Spawn("checkout", "ActiveState-CLI/Reset#"+commitID, ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	contents := fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	contents = bytes.Replace(contents, []byte("3a2d095d-efd6-4be0-b824-21de94fc4ad6"), []byte(""), 1)
	err := fileutils.WriteFile(filepath.Join(ts.Dirs.Work, constants.ConfigFileName), contents)
	suite.Require().NoError(err)

	cp = ts.Spawn("install", "requests")
	cp.Expect("Invalid value")
	cp.Expect("Please run 'state reset' to fix it.")
	cp.ExpectNotExitCode(0)

	cp = ts.Spawn("reset", "-n")
	cp.Expect("Successfully reset to commit: " + commitID)
	cp.ExpectExitCode(0)
}

func TestResetIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ResetIntegrationTestSuite))
}
