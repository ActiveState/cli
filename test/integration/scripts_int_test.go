package integration

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type ScriptsIntegrationTestSuite struct {
	suite.Suite
}

func (suite *ScriptsIntegrationTestSuite) setupConfigFile(ts *e2e.Session) {
	configFileContent := strings.TrimSpace(`
project: "https://platform.activestate.com/ScriptOrg/ScriptProject?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: first-script
    value: echo "first script"
    constraints:
      os: macos,linux
  - name: first-script
    value: echo first script
    constraints:
      os: windows
  - name: second-script
    value: print("second script")
    language: python3
`)

	ts.PrepareActiveStateYAML(configFileContent)
}

func (suite *ScriptsIntegrationTestSuite) TestScripts_EditorV0() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.setupConfigFile(ts)

	cp := ts.Spawn("scripts", "--output", "editor.v0")
	cp.Expect(`[{"name":"first-script"},{"name":"second-script"}]`)
	cp.ExpectExitCode(0)
}

func TestScriptsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ScriptsIntegrationTestSuite))
}
