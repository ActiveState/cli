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
	"github.com/ActiveState/cli/pkg/localcommit"
)

type ResetIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ResetIntegrationTestSuite) TestReset() {
	suite.OnlyRunForTags(tagsuite.Reset)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()
	commitID, err := localcommit.Get(ts.Dirs.Work)
	suite.Require().NoError(err)

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "shared/zlib")
	cp.Expect("Package added")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("history")
	cp.Expect("zlib")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("reset")
	cp.Expect("Your project will be reset to " + commitID.String())
	cp.Expect("Are you sure")
	cp.Expect("(y/N)")
	cp.SendLine("y")
	cp.Expect("Successfully reset to commit: " + commitID.String())
	cp.ExpectExitCode(0)

	cp = ts.Spawn("history")
	cp.ExpectExitCode(0)
	suite.Assert().NotContains(cp.Snapshot(), "zlib")

	cp = ts.Spawn("reset")
	cp.Expect("You are already on the latest commit")
	cp.ExpectNotExitCode(0)

	cp = ts.Spawn("reset", "00000000-0000-0000-0000-000000000000")
	cp.Expect("The given commit ID does not exist")
	cp.ExpectNotExitCode(0)
}

func (suite *ResetIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Reset, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	cp := ts.Spawn("reset", "265f9914-ad4d-4e0a-a128-9d4e8c5db820", "-o", "json")
	cp.Expect(`{"commitID":"265f9914-ad4d-4e0a-a128-9d4e8c5db820"}`)
	cp.ExpectExitCode(0)
}

func (suite *ResetIntegrationTestSuite) TestRevertInvalidURL() {
	suite.OnlyRunForTags(tagsuite.Reset)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()
	commitID, err := localcommit.Get(ts.Dirs.Work)
	suite.Require().NoError(err)

	contents := fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	contents = bytes.Replace(contents, []byte(commitID.String()), []byte(""), 1)
	err = fileutils.WriteFile(filepath.Join(ts.Dirs.Work, constants.ConfigFileName), contents)
	suite.Require().NoError(err)

	cp := ts.Spawn("install", "language/python/requests")
	cp.Expect("invalid commit ID")
	cp.Expect("Please run 'state reset' to fix it.")
	cp.ExpectNotExitCode(0)

	cp = ts.Spawn("reset", "-n")
	cp.Expect("Successfully reset to commit: " + commitID.String())
	cp.ExpectExitCode(0)
}

func TestResetIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ResetIntegrationTestSuite))
}
