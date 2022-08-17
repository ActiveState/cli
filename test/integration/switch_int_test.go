package integration

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

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

	suite.PrepareActiveStateYAML(ts, "ActiveState-CLI", "Branches")
	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)

	cp := ts.SpawnWithOpts(e2e.WithArgs("pull"), e2e.AppendEnv(constants.DisableRuntime+"=false"))
	cp.ExpectLongString("Your project in the activestate.yaml has been updated")
	cp.ExpectExitCode(0)

	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if pjfile.BranchName() != "main" {
		suite.FailNow("branch was not set to 'main' after pull")
	}
	mainBranchCommitID := pjfile.CommitID()

	cp = ts.SpawnWithOpts(e2e.WithArgs("switch", "secondbranch"), e2e.AppendEnv(constants.DisableRuntime+"=false"))
	cp.Expect("Updating Runtime")
	cp.ExpectLongString("Successfully switched to branch: secondbranch", 60*time.Second)
	cp.ExpectExitCode(0)

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

	suite.PrepareActiveStateYAML(ts, "ActiveState-CLI", "History")
	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)

	cp := ts.SpawnWithOpts(e2e.WithArgs("pull"), e2e.AppendEnv(constants.DisableRuntime+"=false"))
	cp.ExpectLongString("Your project in the activestate.yaml has been updated")
	cp.ExpectExitCode(0)

	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if pjfile.BranchName() != "main" {
		suite.FailNow("branch was not set to 'main' after pull")
	}
	orignalCommitID := pjfile.CommitID()

	cp = ts.SpawnWithOpts(e2e.WithArgs("switch", "efce7c7a-c61a-4b04-bb00-f8e7edfd247f"), e2e.AppendEnv(constants.DisableRuntime+"=false"))
	cp.Expect("Updating Runtime")
	cp.ExpectLongString("Successfully switched to commit:", 60*time.Second)
	cp.ExpectExitCode(0)

	// Check that branch and commitID were updated
	pjfile, err = projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if pjfile.CommitID() == orignalCommitID {
		suite.FailNow("commitID was not updated after switching branches")
	}
}

func (suite *SwitchIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session, username, project string) {
	asyData := fmt.Sprintf(`project: "https://platform.activestate.com/%s/%s"`, username, project)
	ts.PrepareActiveStateYAML(asyData)
}

func TestSwitchIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(SwitchIntegrationTestSuite))
}
