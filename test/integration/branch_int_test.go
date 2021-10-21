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
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type BranchIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *BranchIntegrationTestSuite) TestBranch_List() {
	suite.OnlyRunForTags(tagsuite.Branches)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts, "ActiveState-CLI", "Branches")

	cp := ts.Spawn("branch")
	expected := `main (Current)
 ├─ firstbranch
 │  └─ firstbranchchild
 │     └─ childoffirstbranchchild
 ├─ secondbranch
 └─ thirdbranch
`
	cp.ExpectLongString(expected)
	cp.ExpectExitCode(0)
}

func (suite *BranchIntegrationTestSuite) TestBranch_Add() {
	suite.OnlyRunForTags(tagsuite.Branches)
	suite.T().Skip("Skip test as state branch add functionality is currently disabled")
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts, e2e.PersistentUsername, "Branch")

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("pull")
	cp.ExpectLongString("Your project in the activestate.yaml has been updated")
	cp.ExpectExitCode(0)

	branchName, err := uuid.NewRandom()
	suite.Require().NoError(err)

	cp = ts.Spawn("branch", "add", branchName.String())
	cp.ExpectExitCode(0)

	cp = ts.Spawn("branch")
	cp.Expect(branchName.String())
	cp.ExpectExitCode(0)
}

func (suite *BranchIntegrationTestSuite) TestBranch_Switch() {
	suite.OnlyRunForTags(tagsuite.Branches)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	err := ts.ClearCache()
	suite.Require().NoError(err)

	suite.PrepareActiveStateYAML(ts, "ActiveState-CLI", "Branches")
	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)

	cp := ts.Spawn("pull")
	cp.ExpectLongString("Your project in the activestate.yaml has been updated")
	cp.ExpectExitCode(0)

	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if pjfile.BranchName() != "main" {
		suite.FailNow("branch was not set to 'main' after pull")
	}
	mainBranchCommitID := pjfile.CommitID()

	cp = ts.Spawn("branch", "switch", "secondbranch")
	cp.Expect("Updating Runtime")
	cp.Expect("Downloading missing artifacts", 60*time.Second)
	cp.Expect("Updating missing artifacts")
	cp.Expect("Installing")
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

func (suite *BranchIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session, username, project string) {
	asyData := fmt.Sprintf(`project: "https://platform.activestate.com/%s/%s"`, username, project)
	ts.PrepareActiveStateYAML(asyData)
}

func TestBranchIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BranchIntegrationTestSuite))
}
