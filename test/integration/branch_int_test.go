package integration

import (
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/termtest"

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

	ts.PrepareProject("ActiveState-CLI/Branches", "35af7414-b44b-4fd7-aa93-2ecad337ed2b")

	cp := ts.SpawnWithOpts(e2e.OptArgs("branch"))
	// Sometimes there's a space before the line break, unsure exactly why, but hence the regex
	cp.ExpectRe(`main \(Current\)\s?\n  ├─ firstbranch\s?\n  │  └─ firstbranchchild\s?\n  │     └─ childoffirstbranchchild\s?\n  ├─ secondbranch\s?\n  └─ thirdbranch`, termtest.OptExpectTimeout(5*time.Second))
	cp.Expect("To switch to another branch,")
	cp.ExpectExitCode(0)

	ts.PrepareProject("ActiveState-CLI/small-python", e2e.CommitIDNotChecked)
	cp = ts.Spawn("branch")
	cp.Expect("main")
	suite.Assert().NotContains(cp.Snapshot(), "To switch to another branch,") // only shows when multiple branches are listed
	cp.ExpectExitCode(0)
}

func (suite *BranchIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Branches, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Branches", e2e.CommitIDNotChecked)

	cp := ts.Spawn("branch", "-o", "json")
	cp.Expect(`"branchID":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestBranchIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BranchIntegrationTestSuite))
}
