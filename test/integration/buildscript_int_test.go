package integration

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type BuildScriptIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *BuildScriptIntegrationTestSuite) TestBuildScript_NeedsReset() {
	suite.OnlyRunForTags(tagsuite.BuildScripts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(fmt.Sprintf("project: https://%s/%s?commitID=%s\nconfig_version: %d\n",
		constants.DefaultAPIHost, "ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b", projectfile.ConfigVersion))

	cp := ts.Spawn("config", "set", constants.OptinBuildscriptsConfig, "true")
	cp.ExpectExitCode(0)

	suite.Require().NoFileExists(filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName))

	cp = ts.SpawnWithOpts(e2e.OptArgs("refresh"), e2e.OptAppendEnv(constants.DisableRuntime+"=false"))
	cp.Expect("Your project is missing its buildscript file")
	cp.ExpectExitCode(1)

	cp = ts.SpawnWithOpts(e2e.OptArgs("reset", "LOCAL"), e2e.OptAppendEnv(constants.DisableRuntime+"=false"))
	cp.ExpectExitCode(0, e2e.RuntimeSourcingTimeoutOpt)

	suite.Require().FileExists(filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName), ts.DebugMessage(""))
}

func TestBuildScriptIntegrationTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(BuildScriptIntegrationTestSuite))
}
