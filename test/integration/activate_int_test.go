package integration

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type ActivateIntegrationTestSuite struct {
	integration.Suite
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython3() {
	suite.activatePython("3")
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython2() {
	suite.T().Skip("Python 2 is not officially supported by the platform ATM.")
	suite.activatePython("2")
}

func (suite *ActivateIntegrationTestSuite) TestActivateWithoutRuntime() {

	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("activate_test_no_runtime")
	defer cb()

	suite.LoginAsPersistentUser()

	suite.Spawn("activate", "ActiveState-CLI/Python3")
	suite.Expect("Where would you like to checkout")
	suite.SendLine(tempDir)
	suite.Expect("activated state", 20*time.Second)
	suite.WaitForInput(10 * time.Second)

	suite.SendLine("exit 123")
	suite.ExpectExitCode(123)
}

func (suite *ActivateIntegrationTestSuite) activatePython(version string) {
	if runtime.GOOS == "darwin" {
		suite.T().Skip("Runtimes are not supported on macOS")
	}
	if runtime.GOOS == "windows" {
		suite.T().Skip("suite.AppendEnv() does not work on windows currently.  Skipping this test.")
	}

	pythonExe := "python" + version

	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("activate_test")
	defer cb()

	suite.LoginAsPersistentUser()
	suite.AppendEnv([]string{"ACTIVESTATE_CLI_DISABLE_RUNTIME=false"})

	suite.Spawn("activate", "ActiveState-CLI/Python"+version)
	suite.Expect("Where would you like to checkout")
	suite.SendLine(tempDir)
	suite.Expect("Downloading", 20*time.Second)
	suite.Expect("Installing", 120*time.Second)
	suite.Expect("activated state", 120*time.Second)

	// ensure that terminal contains output "Installing x/y" with x, y numbers and x=y
	installingString := regexp.MustCompile(
		"Installing *([0-9]+) */ *([0-9]+)",
	).FindAllStringSubmatch(suite.TerminalSnapshot(), 1)
	suite.Require().Len(installingString, 1, "no match for Installing x / x in\n%s", suite.TerminalSnapshot())
	suite.Require().Equalf(
		installingString[0][1], installingString[0][2],
		"expected all artifacts are reported to be installed, got %s", installingString[0][0],
	)

	// ensure that shell is functional
	suite.WaitForInput()

	// test python
	suite.SendLine(pythonExe + " -c \"import sys; print(sys.copyright)\"")
	suite.Expect("ActiveState Software Inc.")
	suite.SendLine(pythonExe + " -c \"import pytest; print(pytest.__doc__)\"")
	suite.Expect("unit and functional testing")

	// de-activate shell
	suite.SendLine("exit")
	suite.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython3_Forward() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("activate_test_forward")
	defer cb()
	suite.SetWd(tempDir)

	projectFile := &projectfile.Project{}
	contents := strings.TrimSpace(fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState-CLI/Python3"
branch: %s
version: %s
`, constants.BranchName, constants.Version))

	err := yaml.Unmarshal([]byte(contents), projectFile)
	suite.Require().NoError(err)

	projectFile.SetPath(filepath.Join(tempDir, "activestate.yaml"))
	fail := projectFile.Save()
	suite.Require().NoError(fail.ToError())
	suite.Require().FileExists(filepath.Join(tempDir, "activestate.yaml"))

	suite.LoginAsPersistentUser()
	suite.AppendEnv([]string{"ACTIVESTATE_CLI_DISABLE_RUNTIME=false"})

	// Ensure we have the most up to date version of the project before activating
	suite.Spawn("pull")
	suite.Expect("Your activestate.yaml has been updated to the latest version available")
	suite.Expect("Please reactivate any activated instances of the State Tool")
	suite.ExpectExitCode(0)

	suite.Spawn("activate")
	suite.Expect("Activating state: ActiveState-CLI/Python3")

	// not waiting for activation, as we test that part in a different test
	suite.WaitForInput()
	suite.SendLine("exit")
	suite.ExpectExitCode(0)
}

func (suite *ActivateIntegrationTestSuite) TestActivate_Output() {
	tempDir, cb := suite.PrepareTemporaryWorkingDirectory("activate_test")
	defer cb()

	suite.LoginAsPersistentUser()
	suite.Spawn("activate", "ActiveState-CLI/Python3", "--output", "json")
	suite.Expect("Where would you like to checkout")
	suite.SendLine(tempDir)
	suite.Expect("[activated-JSON]")
	suite.Wait()
	suite.Contains(suite.Output(), "ACTIVESTATE_PROJECT")
}

func TestActivateIntegrationTestSuite(t *testing.T) {
	_ = suite.Run // vscode won't show test helpers unless I use this .. -.-

	//suite.Run(t, new(ActivateIntegrationTestSuite))
	integration.RunParallel(t, new(ActivateIntegrationTestSuite))
}
