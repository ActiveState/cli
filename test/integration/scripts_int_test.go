package integration

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

type ScriptsIntegrationTestSuite struct {
	integration.Suite
	cleanup func()
}

func (suite *ScriptsIntegrationTestSuite) SetupTest() {
	suite.Suite.SetupTest()

	var tempDir string
	tempDir, suite.cleanup = suite.PrepareTemporaryWorkingDirectory("ScriptsIntegrationTestSuite")

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

	projectFile := &projectfile.Project{}
	err := yaml.Unmarshal([]byte(configFileContent), projectFile)
	suite.Require().NoError(err)

	fmt.Println("config filepath: ", filepath.Join(tempDir, constants.ConfigFileName))
	projectFile.SetPath(filepath.Join(tempDir, constants.ConfigFileName))
	fail := projectFile.Save()
	suite.Require().NoError(fail.ToError())
}

func (suite *ScriptsIntegrationTestSuite) TearDownTest() {
	suite.Suite.TearDownTest()
	suite.cleanup()
}

func TestScriptsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ScriptsIntegrationTestSuite))
}
