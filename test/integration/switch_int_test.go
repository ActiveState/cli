package integration

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/runbits/commitmediator"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
)

type SwitchIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *SwitchIntegrationTestSuite) TestSwitch_Branch() {
	suite.OnlyRunForTags(tagsuite.Switch)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	err := ts.ClearCache()
	suite.Require().NoError(err)

	ts.PrepareProject("ActiveState-CLI/Branches", "b5b327f8-468e-4999-a23e-8bee886e6b6d")
	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)

	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	suite.Require().Equal("main", pjfile.BranchName(), "branch was not set to 'main' after pull")
	mainBranchCommitID, err := commitmediator.Get(pjfile)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(e2e.OptArgs("switch", "secondbranch"))
	cp.Expect("Operating on project")
	cp.Expect("ActiveState-CLI/Branches")
	cp.Expect("Successfully switched to branch:")
	if runtime.GOOS != "windows" {
		cp.ExpectExitCode(0)
	}

	// Check that branch and commitID were updated
	pjfile, err = projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	commitID, err := commitmediator.Get(pjfile)
	suite.Require().NoError(err)
	suite.Require().NotEqual(mainBranchCommitID, commitID, "commitID was not updated after switching branches")
	suite.Require().Equal("secondbranch", pjfile.BranchName(), "branch was not updated after switching branches")
}

func (suite *SwitchIntegrationTestSuite) TestSwitch_CommitID() {
	suite.OnlyRunForTags(tagsuite.Switch)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	err := ts.ClearCache()
	suite.Require().NoError(err)

	ts.PrepareProject("ActiveState-CLI/History", "b5b327f8-468e-4999-a23e-8bee886e6b6d")
	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)

	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	suite.Require().Equal("main", pjfile.BranchName(), "branch was not set to 'main' after pull")
	originalCommitID, err := commitmediator.Get(pjfile)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(e2e.OptArgs("switch", "efce7c7a-c61a-4b04-bb00-f8e7edfd247f"))
	cp.Expect("Successfully switched to commit:")
	if runtime.GOOS != "windows" {
		cp.ExpectExitCode(0)
	}

	// Check that branch and commitID were updated
	pjfile, err = projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	commitID, err := commitmediator.Get(pjfile)
	suite.Require().NotEqual(originalCommitID, commitID, "commitID was not updated after switching branches")
}

func (suite *SwitchIntegrationTestSuite) TestSwitch_CommitID_NotInHistory() {
	suite.OnlyRunForTags(tagsuite.Switch)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	err := ts.ClearCache()
	suite.Require().NoError(err)

	ts.PrepareProject("ActiveState-CLI/History", "b5b327f8-468e-4999-a23e-8bee886e6b6d")
	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)

	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	suite.Require().Equal("main", pjfile.BranchName(), "branch was not set to 'main' after pull")
	originalCommitID, err := commitmediator.Get(pjfile)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(e2e.OptArgs("switch", "76dff77a-66b9-43e3-90be-dc75917dd661"))
	cp.Expect("Commit does not belong")
	if runtime.GOOS != "windows" {
		cp.ExpectExitCode(1)
		ts.IgnoreLogErrors()
	}

	// Check that branch and commitID were not updated
	pjfile, err = projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	commitID, err := commitmediator.Get(pjfile)
	suite.Require().NoError(err)
	suite.Equal(originalCommitID, commitID, "commitID was updated after switching branches")
}

func (suite *SwitchIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Switch, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Branches", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("switch", "firstbranch", "--output", "json")
	cp.Expect(`"branch":`)
	cp.ExpectExitCode(0)
	// AssertValidJSON(suite.T(), cp) // cannot assert here due to "Skipping runtime setup" notice
}

func TestSwitchIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SwitchIntegrationTestSuite))
}
