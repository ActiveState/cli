package integration

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ScriptsIntegrationTestSuite struct {
	tagsuite.Suite
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
  - name: super-script
    language: bash
    value: |
      $scripts.first-script.path._posix()
  - name: testenv
    language: bash
    value: echo $I_SHOULD_EXIST
`)

	ts.PrepareActiveStateYAML(configFileContent)
}

func (suite *ScriptsIntegrationTestSuite) TestRunInheritEnv() {
	suite.OnlyRunForTags(tagsuite.Scripts)
	ts := e2e.New(suite.T(), false)
	suite.setupConfigFile(ts)

	cp := ts.SpawnWithOpts(e2e.WithArgs("run", "testenv"), e2e.AppendEnv("I_SHOULD_EXIST=I_SURE_DO_EXIST"))
	cp.Expect("I_SURE_DO_EXIST")
	cp.ExpectExitCode(0)
}

func (suite *ScriptsIntegrationTestSuite) TestRunSubscripts() {
	suite.OnlyRunForTags(tagsuite.Scripts)
	ts := e2e.New(suite.T(), false)
	suite.setupConfigFile(ts)

	cp := ts.Spawn("run", "super-script")
	cp.Expect("first script")
	cp.ExpectExitCode(0)
}

func TestScriptsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ScriptsIntegrationTestSuite))
}
