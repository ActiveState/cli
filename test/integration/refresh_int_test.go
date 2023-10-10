package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/ActiveState/termtest"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
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
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Setting Up Runtime")
	cp.Expect("Runtime updated", termtest.OptExpectTimeout(180*time.Second))
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("exec", "--", "python3", "-c", "import requests"),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("ModuleNotFoundError")
	cp.ExpectExitCode(1)

	suite.PrepareActiveStateYAML(ts, "ActiveState-CLI/Branches", "secondbranch", "46c83477-d580-43e2-a0c6-f5d3677517f1")
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("refresh"),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Setting Up Runtime")
	cp.Expect("Runtime updated", termtest.OptExpectTimeout(180*time.Second))
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("exec", "--", "python3", "-c", "import requests"),
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.ExpectExitCode(0)

	cp = ts.Spawn("refresh")
	suite.Assert().NotContains(cp.Output(), "Setting Up Runtime", "Unchanged runtime should not refresh")
	cp.Expect("Runtime updated", termtest.OptExpectTimeout(180*time.Second))
	cp.ExpectExitCode(0)
}

func (suite *RefreshIntegrationTestSuite) NoTestJSON() {
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
