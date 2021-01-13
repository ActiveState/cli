package integration

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type BranchIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *BranchIntegrationTestSuite) TestBranch_List() {
	suite.OnlyRunForTags(tagsuite.Branches)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("branch")
	expectations := []string{
		"firstbranchchild",
		"secondbranch",
		"firstbranch",
		"main",
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

	username := ts.CreateNewUser()
	namespace := fmt.Sprintf("%s/%s", username, "branch-test")

	cp := ts.Spawn("fork", "ActiveState-CLI/Platforms", "--org", username, "--name", "platform-test")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("activate", namespace, "--path="+ts.Dirs.Work, "--output=json")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("branch", "add", "another-branch")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("branch")
	cp.Expect("another-branch")
	cp.ExpectExitCode(0)
}

func (suite *BranchIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	asyData := `project: "https://platform.activestate.com/ActiveState-CLI/Branches"`
	ts.PrepareActiveStateYAML(asyData)
}

func TestBranchIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BranchIntegrationTestSuite))
}
