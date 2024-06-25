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
		constants.DefaultAPIHost, "ActiveState-CLI/Empty", "6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8", projectfile.ConfigVersion))

	cp := ts.Spawn("config", "set", constants.OptinBuildscriptsConfig, "true")
	cp.ExpectExitCode(0)

	suite.Require().NoFileExists(filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName))

	cp = ts.Spawn("refresh")
	cp.Expect("Your project is missing its buildscript file")
	cp.ExpectExitCode(1)

	cp = ts.Spawn("reset", "LOCAL")
	cp.ExpectExitCode(0)

	suite.Require().FileExists(filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName), ts.DebugMessage(""))
}

func TestBuildScriptIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BuildScriptIntegrationTestSuite))
}
