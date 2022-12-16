package integration

import (
	"fmt"
	"path/filepath"

	"github.com/ActiveState/cli/internal-as/testhelpers/e2e"
	"github.com/ActiveState/cli/internal-as/testhelpers/tagsuite"
)

type RevertIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *RevertIntegrationTestSuite) TestRevert_failsOnCommitNotInHistory() {
	suite.OnlyRunForTags(tagsuite.Revert)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()
	username := e2e.PersistentUsername
	project := "small-python"
	namespace := fmt.Sprintf("%s/%s", username, project)

	cp := ts.SpawnWithOpts(e2e.WithArgs("checkout", namespace))
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	wd := filepath.Join(ts.Dirs.Work, namespace)
	// valid commit id not from project
	commitID := "cb9b1aab-8e40-4a1d-8ad6-5ea112da40f1" // from Perl-5.32

	cp = ts.SpawnWithOpts(e2e.WithArgs("revert", commitID), e2e.WithWorkDirectory(wd))
	cp.SendLine("Y")
	cp.Expect(commitID)
	cp.Expect("The target commit is not")
	cp.ExpectNotExitCode(0)
}
