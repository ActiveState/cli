package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
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
	suite.Expect("activated state", 20*time.Second) // Note this line is REQUIRED. For reasons I cannot figure out the below WaitForInput will fail unless the subshell prints something.
	suite.WaitForInput(10 * time.Second)

	f, err := os.Create(path.Join(tempDir, "activestate.yaml"))
	suite.Require().NoError(err)
	f.WriteString("project: https://platform.activestate.com/Owner/ProjectName\n" +
		"scripts:\n" +
		"   - name: test\n" +
		"     description: A script that runs for 20 seconds doing nothing.  It should be interrupted.\n" +
		"     value: |\n" +
		"          echo start of script\n" +
		"          timeout 4\n" +
		"          echo ONLY PRINT IF NOT INTERRUPTED\n" +
		"     constraints:\n" +
		"       os: windows\n" +
		"   - name: test\n" +
		"     description: A script that runs for 20 seconds doing nothing.  It should be interrupted.\n" +
		"     value: |\n" +
		"          echo start of script\n" +
		"          sleep 4\n" +
		"          echo ONLY PRINT IF NOT INTERRUPTED\n" +
		"     constraints:\n" +
		"       os: linux,macos\n")
	f.Close()

	suite.SendLine(fmt.Sprintf("%s run test", suite.Executable()))
	suite.Expect("start of script", 5*time.Second)
	suite.Send(string([]byte{0x3}))
	suite.Expect("^C")
	suite.SendLine("exit")
	suite.Wait()
	suite.Require().NotContains(
		suite.TerminalSnapshot(), "ONLY PRINT IF NOT INTERRUPTED",
	)
	suite.Require().NotContains(
		suite.TerminalSnapshot(), "Terminate batch job",
	)
}

func (suite *ActivateIntegrationTestSuite) activatePython(version string) {
	defer suite.TeardownTest()
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
	suite.Wait()
}

func TestActivateIntegrationTestSuite(t *testing.T) {
	_ = suite.Run // vscode won't show test helpers unless I use this .. -.-

	//suite.Run(t, new(ActivateIntegrationTestSuite))
	integration.RunParallel(t, new(ActivateIntegrationTestSuite))
}
