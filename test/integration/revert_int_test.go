package integration

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal-as/testhelpers/e2e"
	"github.com/ActiveState/cli/internal-as/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type RevertIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *RevertIntegrationTestSuite) TestRevert() {
	suite.OnlyRunForTags(tagsuite.Revert)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.LoginAsPersistentUser()

	namespace := "activestate-cli/Revert"
	cp := ts.SpawnWithOpts(e2e.WithArgs("checkout", namespace))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)
	wd := filepath.Join(ts.Dirs.Work, "Revert")

	// Revert the commit that added urllib3.
	commitID := "1f4f4f7d-7883-400e-b2ad-a5803c018ecd"
	cp = ts.SpawnWithOpts(e2e.WithArgs("revert", commitID), e2e.WithWorkDirectory(wd))
	cp.ExpectLongString(fmt.Sprintf("Operating on project %s", namespace))
	cp.SendLine("Y")
	cp.Expect("You are about to revert the following commit:")
	cp.Expect(commitID)
	cp.Expect("Successfully reverted commit:")
	cp.ExpectExitCode(0)

	// Verify the commit history has both the new revert commit and all prior history.
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("history"),
		e2e.WithWorkDirectory(wd),
	)
	cp.Expect("Revert commit " + commitID)
	cp.Expect("- urllib3")
	cp.Expect("+ argparse") // parent commit
	cp.Expect("+ urllib3")  // commit whose changes were just reverted
	cp.Expect("+ python")   // initial commit

	// Verify that argparse still exists (it was not reverted along with urllib3).
	cp = ts.SpawnWithOpts(
		e2e.WithArgs("shell", "Revert"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.WaitForInput()
	cp.SendLine("python3")
	cp.Expect("3.9.15")
	cp.SendLine("import urllib3")
	cp.Expect("No module named 'urllib3'")
	cp.SendLine("import argparse")
	suite.Assert().NotContains(cp.TrimmedSnapshot(), "No module named 'argparse'")
	cp.SendLine("exit()") // exit python3
	cp.SendLine("exit")   // exit state shell
	cp.ExpectExitCode(0)
}

func (suite *RevertIntegrationTestSuite) TestRevert_failsOnCommitNotInHistory() {
	suite.OnlyRunForTags(tagsuite.Revert)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	namespace := "activestate-cli/small-python"
	cp := ts.SpawnWithOpts(e2e.WithArgs("checkout", namespace))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)
	wd := filepath.Join(ts.Dirs.Work, "small-python")

	// valid commit id not from project
	commitID := "cb9b1aab-8e40-4a1d-8ad6-5ea112da40f1" // from Perl-5.32
	cp = ts.SpawnWithOpts(e2e.WithArgs("revert", commitID), e2e.WithWorkDirectory(wd))
	cp.ExpectLongString(fmt.Sprintf("Operating on project %s", namespace))
	cp.SendLine("Y")
	cp.Expect(commitID)
	cp.ExpectLongString("The commit being reverted is not within the current commit's history")
	cp.ExpectNotExitCode(0)
}

func TestRevertIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RevertIntegrationTestSuite))
}
