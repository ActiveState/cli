package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ExportIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ExportIntegrationTestSuite) TestExport_Export() {
	suite.OnlyRunForTags("export")
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)
	cp := ts.Spawn("export", "recipe")
	cp.Expect("{\"camel_flags\":")
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_ExportArg() {
	suite.OnlyRunForTags("export")
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)
	cp := ts.Spawn("export", "recipe")
	cp.Expect("{\"camel_flags\":")
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_ExportPlatform() {
	suite.OnlyRunForTags("export")
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)
	cp := ts.Spawn("export", "recipe", "--platform", "linux")
	cp.Expect("{\"camel_flags\":")
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_InvalidPlatform() {
	suite.OnlyRunForTags("export")
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)
	cp := ts.Spawn("export", "recipe", "--platform", "junk")
	cp.ExpectExitCode(1)
}

func (suite *ExportIntegrationTestSuite) TestExport_ConfigDir() {
	suite.OnlyRunForTags("export")
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)
	cp := ts.Spawn("export", "config", "--filter", "junk")
	cp.ExpectExitCode(1)
}

func (suite *ExportIntegrationTestSuite) TestExport_Config() {
	suite.OnlyRunForTags("export")
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)
	cp := ts.Spawn("export", "config")
	cp.Expect(`dir: `)
	cp.ExpectExitCode(0)
}

func TestExportIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ExportIntegrationTestSuite))
}

func (suite *ExportIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	asyData := `project: "https://platform.activestate.com/cli-integration-tests/Export"`
	ts.PrepareActiveStateYAML(asyData)
}
