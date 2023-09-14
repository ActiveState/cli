package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
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

	suite.PrepareActiveStateYAML(ts, "ActiveState-CLI", "Branches", "b5b327f8-468e-4999-a23e-8bee886e6b6d")
	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)

	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if pjfile.BranchName() != "main" {
		suite.FailNow("branch was not set to 'main' after pull")
	}
	mainBranchCommitID := pjfile.CommitID()

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
	if pjfile.CommitID() == mainBranchCommitID {
		suite.FailNow("commitID was not updated after switching branches")
	}
	if pjfile.BranchName() != "secondbranch" {
		suite.FailNow("branch was not updated after switching branches")
	}
}

func (suite *SwitchIntegrationTestSuite) TestSwitch_CommitID() {
	suite.OnlyRunForTags(tagsuite.Switch)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	err := ts.ClearCache()
	suite.Require().NoError(err)

	suite.PrepareActiveStateYAML(ts, "ActiveState-CLI", "History", "b5b327f8-468e-4999-a23e-8bee886e6b6d")
	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)

	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if pjfile.BranchName() != "main" {
		suite.FailNow("branch was not set to 'main' after pull")
	}
	orignalCommitID := pjfile.CommitID()

	cp := ts.SpawnWithOpts(e2e.OptArgs("switch", "efce7c7a-c61a-4b04-bb00-f8e7edfd247f"))
	cp.Expect("Successfully switched to commit:")
	if runtime.GOOS != "windows" {
		cp.ExpectExitCode(0)
	}

	// Check that branch and commitID were updated
	pjfile, err = projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if pjfile.CommitID() == orignalCommitID {
		suite.FailNow("commitID was not updated after switching branches")
	}
}

func (suite *SwitchIntegrationTestSuite) TestSwitch_CommitID_NotInHistory() {
	suite.OnlyRunForTags(tagsuite.Switch)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	err := ts.ClearCache()
	suite.Require().NoError(err)

	suite.PrepareActiveStateYAML(ts, "ActiveState-CLI", "History", "b5b327f8-468e-4999-a23e-8bee886e6b6d")
	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)

	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if pjfile.BranchName() != "main" {
		suite.FailNow("branch was not set to 'main' after pull")
	}
	orignalCommitID := pjfile.CommitID()

	cp := ts.SpawnWithOpts(e2e.OptArgs("switch", "76dff77a-66b9-43e3-90be-dc75917dd661"))
	cp.Expect("Commit does not belong")
	if runtime.GOOS != "windows" {
		cp.ExpectExitCode(1)
	}

	// Check that branch and commitID were not updated
	pjfile, err = projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if pjfile.CommitID() != orignalCommitID {
		suite.FailNow("commitID was updated after switching branches")
	}
}

func (suite *SwitchIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session, username, project, commitID string) {
	asyData := fmt.Sprintf(`project: "https://platform.activestate.com/%s/%s?branch=main&commitID=%s"`, username, project, commitID)
	ts.PrepareActiveStateYAML(asyData)
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
