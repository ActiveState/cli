package integration

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type RevertIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *RevertIntegrationTestSuite) TestRevert() {
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		suite.T().Skip("macOS ARM wants to link to system gettext for some reason") // DX-3256
	}
	suite.OnlyRunForTags(tagsuite.Revert)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	namespace := "ActiveState-CLI/Revert"
	ts.PrepareProject(namespace, "c9444988-2761-4b39-8c4c-eb5fdaaa8dca")

	// Revert the commit that added urllib3.
	commitID := "d105e865-d12f-4c42-a1a0-6767590d87da"
	cp := ts.Spawn("revert", commitID)
	cp.Expect(fmt.Sprintf("Operating on project %s", namespace))
	cp.Expect("You are about to revert the following commit:")
	cp.Expect(commitID)
	cp.SendLine("y")
	cp.Expect("Successfully reverted commit:", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	// Verify the commit history has both the new revert commit and all prior history.
	cp = ts.Spawn("history")
	cp.Expect("Reverted commit for commit " + commitID)
	cp.Expect("- urllib3")
	cp.Expect("+ argparse") // parent commit
	cp.Expect("+ urllib3")  // commit whose changes were just reverted
	cp.Expect("+ python")   // initial commit

	// Verify that argparse still exists (it was not reverted along with urllib3).
	cp = ts.Spawn("shell", "Revert")
	cp.ExpectInput(e2e.RuntimeSourcingTimeoutOpt)
	cp.SendLine("python3")
	cp.Expect("3.9.15")
	cp.SendLine("import urllib3")
	cp.Expect("No module named 'urllib3'")
	cp.SendLine("import argparse")
	suite.Assert().NotContains(cp.Output(), "No module named 'argparse'")
	cp.SendLine("exit()") // exit python3
	cp.SendLine("exit")   // exit state shell
	cp.ExpectExitCode(0)
}

func (suite *RevertIntegrationTestSuite) TestRevertRemote() {
	suite.OnlyRunForTags(tagsuite.Revert)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Revert", "75ae9c67-df55-4a95-be6f-b7975e5bb523")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "requests")
	cp.Expect("Added: language/python/requests")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("revert", "REMOTE", "--non-interactive")
	cp.Expect("Successfully reverted")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("history")
	cp.Expect("- requests")
	cp.Expect("+ requests")
	cp.ExpectExitCode(0)
}

func (suite *RevertIntegrationTestSuite) TestRevert_failsOnCommitNotInHistory() {
	suite.OnlyRunForTags(tagsuite.Revert)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	// valid commit id not from project
	commitID := "cb9b1aab-8e40-4a1d-8ad6-5ea112da40f1" // from Perl-5.32
	cp := ts.Spawn("revert", commitID)
	cp.Expect("Operating on project ActiveState-CLI/Empty")
	cp.SendLine("Y")
	cp.Expect(commitID)
	cp.Expect("not found")
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()
}

func (suite *RevertIntegrationTestSuite) TestRevertTo() {
	suite.OnlyRunForTags(tagsuite.Revert)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	namespace := "ActiveState-CLI/Revert"
	ts.PrepareProject(namespace, "c9444988-2761-4b39-8c4c-eb5fdaaa8dca")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	// Revert the commit that added urllib3.
	commitID := "d105e865-d12f-4c42-a1a0-6767590d87da"
	cp = ts.Spawn("revert", "--to", commitID)
	cp.Expect(fmt.Sprintf("Operating on project %s", namespace))
	cp.Expect("You are about to revert to the following commit:")
	cp.Expect(commitID)
	cp.SendLine("Y")
	cp.Expect("Successfully reverted to commit:")
	cp.ExpectExitCode(0)

	// Verify the commit history has both the new revert commit and all prior history.
	cp = ts.Spawn("history")
	cp.Expect("Revert to commit " + commitID)
	cp.Expect("- argparse") // effectively reverting previous commit
	cp.Expect("+ argparse") // commit being effectively reverted
	cp.Expect("+ urllib3")  // commit reverted to
	cp.Expect("+ python")   // initial commit
}

func (suite *RevertIntegrationTestSuite) TestRevertTo_failsOnCommitNotInHistory() {
	suite.OnlyRunForTags(tagsuite.Revert)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	// valid commit id not from project
	commitID := "cb9b1aab-8e40-4a1d-8ad6-5ea112da40f1" // from Perl-5.32
	cp := ts.Spawn("revert", "--to", commitID)
	cp.Expect("Operating on project ActiveState-CLI/Empty")
	cp.SendLine("Y")
	cp.Expect(commitID)
	cp.Expect("not found")
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()
}

func (suite *RevertIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Revert, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Revert", "c9444988-2761-4b39-8c4c-eb5fdaaa8dca")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("revert", "--to", "d105e865-d12f-4c42-a1a0-6767590d87da", "-o", "json")
	cp.Expect(`{"current_commit_id":`, e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestRevertIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RevertIntegrationTestSuite))
}
