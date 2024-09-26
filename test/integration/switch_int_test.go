package integration

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/project"
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

	ts.PrepareEmptyProject()
	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)

	pj, err := project.FromPath(pjfilepath)
	suite.Require().NoError(err)
	suite.Require().Equal("main", pj.BranchName(), "branch was not set to 'main' after pull")
	mainBranchCommitID := ts.CommitID()
	suite.Require().NoError(err)

	cp := ts.Spawn("switch", "mingw")
	cp.Expect("Operating on project")
	cp.Expect("ActiveState-CLI/Empty")
	cp.Expect("Successfully switched to branch:")
	if runtime.GOOS != "windows" {
		cp.ExpectExitCode(0)
	}

	// Check that branch and commitID were updated
	pj, err = project.FromPath(pjfilepath)
	suite.Require().NoError(err)
	suite.Require().NoError(err)
	suite.NotEqual(mainBranchCommitID, ts.CommitID(), "commitID was not updated after switching branches", pj.Dir())
	suite.Equal("mingw", pj.BranchName(), "branch was not updated after switching branches")
}

func (suite *SwitchIntegrationTestSuite) TestSwitch_CommitID() {
	suite.OnlyRunForTags(tagsuite.Switch)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	err := ts.ClearCache()
	suite.Require().NoError(err)

	ts.PrepareEmptyProject()
	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)

	pj, err := project.FromPath(pjfilepath)
	suite.Require().NoError(err)
	suite.Require().Equal("main", pj.BranchName(), "branch was not set to 'main' after pull")
	originalCommitID := ts.CommitID()

	cp := ts.SpawnWithOpts(e2e.OptArgs("switch", "265f9914-ad4d-4e0a-a128-9d4e8c5db820"))
	cp.Expect("Successfully switched to commit:")
	if runtime.GOOS != "windows" {
		cp.ExpectExitCode(0)
	}

	// Check that branch and commitID were updated
	_, err = project.FromPath(pjfilepath)
	suite.Require().NoError(err)
	suite.Require().NotEqual(originalCommitID, ts.CommitID(), "commitID was not updated after switching branches")
}

func (suite *SwitchIntegrationTestSuite) TestSwitch_CommitID_NotInHistory() {
	suite.OnlyRunForTags(tagsuite.Switch)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	err := ts.ClearCache()
	suite.Require().NoError(err)

	ts.PrepareEmptyProject()
	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)

	pj, err := project.FromPath(pjfilepath)
	suite.Require().NoError(err)
	suite.Require().Equal("main", pj.BranchName(), "branch was not set to 'main' after pull")
	originalCommitID := ts.CommitID()

	cp := ts.Spawn("switch", "76dff77a-66b9-43e3-90be-dc75917dd661")
	cp.Expect("Commit does not belong")
	if runtime.GOOS != "windows" {
		cp.ExpectExitCode(1)
		ts.IgnoreLogErrors()
	}

	// Check that branch and commitID were not updated
	_, err = project.FromPath(pjfilepath)
	suite.Require().NoError(err)
	suite.Equal(originalCommitID, ts.CommitID(), "commitID was updated after switching branches")
}

func (suite *SwitchIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Switch, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	cp := ts.Spawn("switch", "mingw", "--output", "json")
	cp.Expect(`"branch":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestSwitchIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SwitchIntegrationTestSuite))
}
