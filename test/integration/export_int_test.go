package integration

import (
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ExportIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ExportIntegrationTestSuite) TestExport_Export() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Export", "")
	cp := ts.Spawn("export", "recipe")
	cp.Expect("{\"camel_flags\":")
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_ExportArg() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Export", "")
	cp := ts.Spawn("export", "recipe")
	cp.Expect("{\"camel_flags\":")
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_ExportPlatform() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Export", "")
	cp := ts.Spawn("export", "recipe", "--platform", "linux")
	cp.Expect("{\"camel_flags\":")
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_InvalidPlatform() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Export", "")
	cp := ts.Spawn("export", "recipe", "--platform", "junk")
	cp.ExpectExitCode(1)
}

func (suite *ExportIntegrationTestSuite) TestExport_ConfigDir() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Export", "")
	cp := ts.Spawn("export", "config", "--filter", "junk")
	cp.ExpectExitCode(1)
}

func (suite *ExportIntegrationTestSuite) TestExport_Config() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("cli-integration-tests/Export", "")
	cp := ts.Spawn("export", "config")
	cp.Expect(`dir: `)
	cp.ExpectLongString(ts.Dirs.Config, time.Second)
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_Env() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Export", "5397f645-da8a-4591-b106-9d7fa99545fe")
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("export", "env"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect(`PATH: `)
	cp.ExpectExitCode(0)

	suite.Assert().NotContains(cp.TrimmedSnapshot(), "ACTIVESTATE_ACTIVATED")
}

func (suite *ExportIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Export, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("export", "config", "-o", "json")
	cp.Expect(`{"dir":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.Spawn("checkout", "ActiveState-CLI/small-python", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("export", "env", "-o", "json"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.ExpectExitCode(0)
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
	//AssertValidJSON(suite.T(), cp) // recipe is too large to fit in terminal snapshot
}

func TestExportIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ExportIntegrationTestSuite))
}
