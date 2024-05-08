package integration

import (
	"fmt"
	"path/filepath"
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
	suite.OnlyRunForTags(tagsuite.Revert)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.LoginAsPersistentUser()

	namespace := "ActiveState-CLI/Revert"
	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", namespace))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)
	wd := filepath.Join(ts.Dirs.Work, "Revert")

	// Revert the commit that added urllib3.
	commitID := "1f4f4f7d-7883-400e-b2ad-a5803c018ecd"
	cp = ts.SpawnWithOpts(e2e.OptArgs("revert", commitID), e2e.OptWD(wd))
	cp.Expect(fmt.Sprintf("Operating on project %s", namespace))
	cp.Expect("You are about to revert the following commit:")
	cp.Expect(commitID)
	cp.SendLine("y")
	cp.Expect("Successfully reverted commit:")
	cp.ExpectExitCode(0)

	// Verify the commit history has both the new revert commit and all prior history.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("history"),
		e2e.OptWD(wd),
	)
	cp.Expect("Reverted commit for commit " + commitID)
	cp.Expect("- urllib3")
	cp.Expect("+ argparse") // parent commit
	cp.Expect("+ urllib3")  // commit whose changes were just reverted
	cp.Expect("+ python")   // initial commit

	// Verify that argparse still exists (it was not reverted along with urllib3).
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("shell", "Revert"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
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

	cp := ts.Spawn("checkout", "ActiveState-CLI/Revert", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "requests")
	cp.Expect("Package added")
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

	namespace := "ActiveState-CLI/small-python"
	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", namespace))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)
	wd := filepath.Join(ts.Dirs.Work, "small-python")

	// valid commit id not from project
	commitID := "cb9b1aab-8e40-4a1d-8ad6-5ea112da40f1" // from Perl-5.32
	cp = ts.SpawnWithOpts(e2e.OptArgs("revert", commitID), e2e.OptWD(wd))
	cp.Expect(fmt.Sprintf("Operating on project %s", namespace))
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
	ts.LoginAsPersistentUser()

	namespace := "ActiveState-CLI/Revert"
	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", namespace))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)
	wd := filepath.Join(ts.Dirs.Work, "Revert")

	// Revert the commit that added urllib3.
	commitID := "1f4f4f7d-7883-400e-b2ad-a5803c018ecd"
	cp = ts.SpawnWithOpts(e2e.OptArgs("revert", "--to", commitID), e2e.OptWD(wd))
	cp.Expect(fmt.Sprintf("Operating on project %s", namespace))
	cp.SendLine("Y")
	cp.Expect("You are about to revert to the following commit:")
	cp.Expect(commitID)
	cp.Expect("Successfully reverted to commit:")
	cp.ExpectExitCode(0)

	// Verify the commit history has both the new revert commit and all prior history.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("history"),
		e2e.OptWD(wd),
	)
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

	namespace := "ActiveState-CLI/small-python"
	cp := ts.SpawnWithOpts(e2e.OptArgs("checkout", namespace))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)
	wd := filepath.Join(ts.Dirs.Work, "small-python")

	// valid commit id not from project
	commitID := "cb9b1aab-8e40-4a1d-8ad6-5ea112da40f1" // from Perl-5.32
	cp = ts.SpawnWithOpts(e2e.OptArgs("revert", "--to", commitID), e2e.OptWD(wd))
	cp.Expect(fmt.Sprintf("Operating on project %s", namespace))
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

	cp := ts.Spawn("checkout", "ActiveState-CLI/Revert", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("revert", "--to", "1f4f4f7d-7883-400e-b2ad-a5803c018ecd", "-o", "json")
	cp.Expect(`{"current_commit_id":`)
	cp.ExpectExitCode(0)
	// AssertValidJSON(suite.T(), cp) // cannot assert here due to "Skipping runtime setup" notice
}

func TestRevertIntegrationTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RevertIntegrationTestSuite))
}
