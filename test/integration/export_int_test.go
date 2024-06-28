package integration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/termtest"
)

type ExportIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ExportIntegrationTestSuite) TestExport_ConfigDir() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("export", "config", "--filter", "junk")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *ExportIntegrationTestSuite) TestExport_Config() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("export", "config")
	cp.Expect(`dir: `)
	cp.Expect(ts.Dirs.Config, termtest.OptExpectTimeout(time.Second))
	cp.ExpectExitCode(0)
}

func (suite *ExportIntegrationTestSuite) TestExport_Env() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()
	cp := ts.Spawn("export", "env")
	cp.Expect(`PATH: `)
	cp.ExpectExitCode(0)

	suite.Assert().NotContains(cp.Output(), "ACTIVESTATE_ACTIVATED")
}

func (suite *ExportIntegrationTestSuite) TestExport_Log() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)
	defer ts.ClearCache()

	cp := ts.Spawn("--version")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("export", "log")
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

func (suite *ExportIntegrationTestSuite) TestExport_Runtime() {
	suite.OnlyRunForTags(tagsuite.Export)
	ts := e2e.New(suite.T(), false)

	ts.PrepareEmptyProject()
	cp := ts.Spawn("export", "runtime")
	cp.Expect("Project Path: ")
	cp.Expect("Runtime Path: ")
	cp.Expect("Executables Path: ")
	cp.Expect("Environment Variables:") // intentional lack of trailing space
	cp.Expect(` - PATH: `)
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

	ts.PrepareEmptyProject()

	cp = ts.Spawn("export", "env", "-o", "json")
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	ts.LoginAsPersistentUser()
	cp = ts.Spawn("export", "jwt", "-o", "json")
	cp.Expect(`{"value":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.Spawn("export", "log", "-o", "json")
	cp.Expect(`{"logFile":"`)
	cp.Expect(`.log"}`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.Spawn("export", "runtime", "-o", "json")
	cp.Expect(`{"project":"`)
	cp.Expect(`"}}`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestExportIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ExportIntegrationTestSuite))
}
