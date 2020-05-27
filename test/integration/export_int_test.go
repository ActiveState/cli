package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type ExportIntegrationTestSuite struct {
	suite.Suite
}

func (suite *ExportIntegrationTestSuite) TestExport_Export() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)
	cp := ts.Spawn("export", "recipe")
	cp.Expect("{\"camel_flags\":")
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_ExportArg() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)
	cp := ts.Spawn("export", "recipe")
	cp.Expect("{\"camel_flags\":")
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_ExportPlatform() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)
	cp := ts.Spawn("export", "recipe", "--platform", "linux")
	cp.Expect("{\"camel_flags\":")
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_InvalidPlatform() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)
	cp := ts.Spawn("export", "recipe", "--platform", "junk")
	cp.ExpectExitCode(1)
}

func TestExportIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ExportIntegrationTestSuite))
}

func (suite *ExportIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	asyData := `project: "https://platform.activestate.com/cli-integration-tests/Export"`
	ts.PrepareActiveStateYAML(asyData)
}
