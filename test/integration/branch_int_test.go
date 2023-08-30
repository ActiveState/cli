package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/ActiveState/termtest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type BranchIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *BranchIntegrationTestSuite) TestBranch_List() {
	suite.OnlyRunForTags(tagsuite.Branches)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts, "ActiveState-CLI", "Branches")

	cp := ts.SpawnWithOpts(e2e.OptArgs("branch"))
	cp.Expect(` main (Current)
  ├─ firstbranch
  │  └─ firstbranchchild
  │     └─ childoffirstbranchchild
  ├─ secondbranch
  └─ thirdbranch`, termtest.OptExpectTimeout(5*time.Second))
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
	cp.Expect("Your project in the activestate.yaml has been updated")
	cp.ExpectExitCode(0)

	branchName, err := uuid.NewRandom()
	suite.Require().NoError(err)

	cp = ts.Spawn("branch", "add", branchName.String())
	cp.ExpectExitCode(0)

	cp = ts.Spawn("branch")
	cp.Expect(branchName.String())
	cp.ExpectExitCode(0)
}

func (suite *BranchIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session, username, project string) {
	asyData := fmt.Sprintf(`project: "https://platform.activestate.com/%s/%s"`, username, project)
	ts.PrepareActiveStateYAML(asyData)
}

func (suite *BranchIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Branches, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Branches", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("branch", "-o", "json")
	cp.Expect(`"branchID":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestBranchIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BranchIntegrationTestSuite))
}
