package integration

import (
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type RefreshIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *RefreshIntegrationTestSuite) TestRefresh() {
	suite.OnlyRunForTags(tagsuite.Refresh)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts, "ActiveState-CLI/Branches", "main", "35af7414-b44b-4fd7-aa93-2ecad337ed2b")

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("refresh"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Runtime updated", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("exec", "--", "python3", "-c", "import requests"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("ModuleNotFoundError")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	suite.PrepareActiveStateYAML(ts, "ActiveState-CLI/Branches", "secondbranch", "46c83477-d580-43e2-a0c6-f5d3677517f1")
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("refresh"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Runtime updated", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("exec", "--", "python3", "-c", "import requests"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.ExpectExitCode(0, e2e.RuntimeSourcingTimeoutOpt)

	cp = ts.Spawn("refresh")
	cp.Expect("Runtime updated")
	cp.ExpectExitCode(0)
	suite.Assert().NotContains(cp.Output(), "Installing", "Unchanged runtime should not refresh")
}

func (suite *RefreshIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Refresh, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts, "ActiveState-CLI/Branches", "main", "35af7414-b44b-4fd7-aa93-2ecad337ed2b")

	cp := ts.Spawn("refresh", "-o", "json")
	cp.Expect(`"namespace":`)
	cp.Expect(`"path":`)
	cp.Expect(`"executables":`)
	cp.ExpectExitCode(0)
	// AssertValidJSON(suite.T(), cp) // cannot assert here due to "Skipping runtime setup" notice
}

func (suite *RefreshIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session, namespace, branch, commitID string) {
	asyData := fmt.Sprintf(`project: "https://platform.activestate.com/%s?branch=%s"`, namespace, branch)
	ts.PrepareActiveStateYAML(asyData)
	ts.PrepareCommitIdFile(commitID)
}

func TestRefreshIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(RefreshIntegrationTestSuite))
}
