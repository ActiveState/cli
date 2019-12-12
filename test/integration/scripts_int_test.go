package integration

import (
	"fmt"
	"io/ioutil"
	"os"
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
	originalWd string
}

func (suite *ScriptsIntegrationTestSuite) SetupTest() {
	suite.Suite.SetupTest()

	tempDir, err := ioutil.TempDir("", suite.T().Name())
	suite.Require().NoError(err)

	suite.originalWd, err = os.Getwd()
	suite.Require().NoError(err)
	err = os.Chdir(tempDir)
	suite.Require().NoError(err)

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
	err = yaml.Unmarshal([]byte(configFileContent), projectFile)
	suite.Require().NoError(err)

	fmt.Println("config filepath: ", filepath.Join(tempDir, constants.ConfigFileName))
	projectFile.SetPath(filepath.Join(tempDir, constants.ConfigFileName))
	fail := projectFile.Save()
	suite.Require().NoError(fail.ToError())

	suite.SetWd(tempDir)
}

func (suite *ScriptsIntegrationTestSuite) TearDownTest() {
	suite.Suite.TearDownTest()
	os.Chdir(suite.originalWd)
}

func (suite *ScriptsIntegrationTestSuite) TestScripts_EditorV0() {
	suite.Spawn("scripts", "--output", "editor.v0")
	suite.Expect("[{\"name\":\"first-script\"},{\"name\":\"second-script\"}]")
	suite.Wait()
}

func TestScriptsIntegrationTestSuite(t *testing.T) {
	_ = suite.Run

	integration.RunParallel(t, new(ScriptsIntegrationTestSuite))
}
