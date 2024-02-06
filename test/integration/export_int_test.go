package integration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/suite"
)

type ExportIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ExportIntegrationTestSuite) TestExport_Export() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Export", "efe4433a-4c27-4b13-b27f-da0b3646d98e")
	cp := ts.Spawn("export", "recipe")
	cp.Expect("{\"camel_flags\":")
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_ExportArg() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Export", "efe4433a-4c27-4b13-b27f-da0b3646d98e")
	cp := ts.Spawn("export", "recipe")
	cp.Expect("{\"camel_flags\":")
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_ExportPlatform() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Export", "efe4433a-4c27-4b13-b27f-da0b3646d98e")
	cp := ts.Spawn("export", "recipe", "--platform", "linux")
	cp.Expect("{\"camel_flags\":")
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_InvalidPlatform() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Export", "efe4433a-4c27-4b13-b27f-da0b3646d98e")
	cp := ts.Spawn("export", "recipe", "--platform", "junk")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *ExportIntegrationTestSuite) TestExport_ConfigDir() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Export", e2e.CommitIDNotChecked)
	cp := ts.Spawn("export", "config", "--filter", "junk")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *ExportIntegrationTestSuite) TestExport_Config() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Export", e2e.CommitIDNotChecked)
	cp := ts.Spawn("export", "config")
	cp.Expect(`dir: `)
	cp.Expect(ts.Dirs.Config, termtest.OptExpectTimeout(time.Second))
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_Env() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Export", "5397f645-da8a-4591-b106-9d7fa99545fe")
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("export", "env"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect(`PATH: `, e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	suite.Assert().NotContains(cp.Output(), "ACTIVESTATE_ACTIVATED")
}

func (suite *ExportIntegrationTestSuite) TestLog() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.ClearCache()

	cp := ts.Spawn("export", "log")
	cp.Expect(filepath.Join(ts.Dirs.Config, "logs"))
	cp.ExpectRe(`state-\d+`)
	cp.Expect(".log")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("export", "log", "state-svc")
	cp.Expect(filepath.Join(ts.Dirs.Config, "logs"))
	cp.ExpectRe(`state-svc-\d+`)
	cp.Expect(".log")
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Export, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("export", "config", "-o", "json")
	cp.Expect(`{"dir":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/small-python", "."),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.ExpectExitCode(0, e2e.RuntimeSourcingTimeoutOpt)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("export", "env", "-o", "json"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.ExpectExitCode(0, e2e.RuntimeSourcingTimeoutOpt)
	AssertValidJSON(suite.T(), cp)

	ts.LoginAsPersistentUser()
	cp = ts.Spawn("export", "jwt", "-o", "json")
	cp.Expect(`{"value":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.Spawn("export", "recipe", "-o", "json")
	cp.Expect(`{`)
	cp.Expect(`}`)
	cp.ExpectExitCode(0)
	// AssertValidJSON(suite.T(), cp) // recipe is too large to fit in terminal snapshot

	cp = ts.Spawn("export", "log", "-o", "json")
	cp.Expect(`{"logFile":"`)
	cp.Expect(`.log"}`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestExportIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ExportIntegrationTestSuite))
}
