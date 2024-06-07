package integration

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type MigratorIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *MigratorIntegrationTestSuite) TestMigrator() {
	suite.OnlyRunForTags(tagsuite.Migrations, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp := ts.SpawnWithOpts(e2e.OptArgs("refresh"))
	cp.ExpectExitCode(0)

	suite.Require().Contains(string(fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))),
		fmt.Sprintf("config_version: %d", projectfile.ConfigVersion), ts.DebugMessage(""))
}

func (suite *MigratorIntegrationTestSuite) TestMigrator_Buildscripts() {
	suite.OnlyRunForTags(tagsuite.Migrations, tagsuite.BuildScripts, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("config", "set", constants.OptinBuildscriptsConfig, "true")
	cp.ExpectExitCode(0)

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	suite.Require().NoFileExists(filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName))

	cp = ts.SpawnWithOpts(e2e.OptArgs("refresh"), e2e.OptAppendEnv(constants.DisableRuntime+"=false"))
	cp.ExpectExitCode(0, e2e.RuntimeSourcingTimeoutOpt)

	suite.Require().FileExists(filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName), ts.DebugMessage(""))
}

func TestMigratorIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(MigratorIntegrationTestSuite))
}
