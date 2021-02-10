package integration

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
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
	expectations := []string{
		"firstbranch",
		"firstbranchchild",
		"main",
		"secondbranch",
	}
	for _, expectation := range expectations {
		cp.Expect(expectation)
	}
	cp.ExpectExitCode(0)
}

func (suite *BranchIntegrationTestSuite) TestBranch_Add() {
	suite.OnlyRunForTags(tagsuite.Platforms)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	projectName := "Branch"
	suite.PrepareActiveStateYAML(ts, e2e.PersistentUsername, projectName)

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("pull")
	cp.ExpectLongString("Your project in the activestate.yaml has been updated")
	cp.ExpectExitCode(0)

	cp = ts.SpawnCmd("cat", "activestate.yaml")
	cp.ExpectExitCode(0)
	fmt.Println(cp.TrimmedSnapshot())

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

func TestBranchIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BranchIntegrationTestSuite))
}
