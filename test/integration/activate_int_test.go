package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/ActiveState/cli/pkg/expect"
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
	if runtime.GOOS == "windows" {
		return // See command below on why test on windows does not work right now.
	}

	tempDir, err := ioutil.TempDir("", "activate_test_no_runtime")
	suite.Require().NoError(err)
	err = os.RemoveAll(tempDir)
	suite.Require().NoError(err)
	err = os.MkdirAll(tempDir, 0770)
	suite.Require().NoError(err)
	err = os.Chdir(tempDir)
	suite.Require().NoError(err)

	defer func() {
		os.Chdir(os.TempDir())
		os.RemoveAll(tempDir)
	}()

	suite.LoginAsPersistentUser()

	suite.Spawn("activate", "ActiveState-CLI/Python3")
	suite.Expect("Where would you like to checkout")
	suite.SendLine(tempDir)
	suite.Expect("activated state", 20*time.Second) // Note this line is REQUIRED. For reasons I cannot figure out the below WaitForInput will fail unless the subshell prints something.
	suite.WaitForInput(10 * time.Second)
	suite.SendLine("exit")
	suite.Wait()
}

func (suite *ActivateIntegrationTestSuite) activatePython(version string) {
	// We are currently disabling these tests on Windows, because when the state tool spawns a
	// bash or CMD shell in a ConPTY environment, the text send to the PTY is not forwarded to
	// the shell.  It is unclear, if this can be fixed right now.
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		return // Runtimes aren't supported on macOS
	}

	pythonExe := "python" + version

	tempDir, err := ioutil.TempDir("", "activate_test")
	fmt.Printf("temporary directory is: %s\n", tempDir)
	suite.Require().NoError(err)
	err = os.RemoveAll(tempDir)
	suite.Require().NoError(err)
	err = os.MkdirAll(tempDir, 0770)
	suite.Require().NoError(err)
	err = os.Chdir(tempDir)
	suite.Require().NoError(err)

	defer func() {
		os.Chdir(os.TempDir())
		os.RemoveAll(tempDir)
	}()

	suite.LoginAsPersistentUser()
	suite.AppendEnv([]string{"ACTIVESTATE_CLI_DISABLE_RUNTIME=false"})

	suite.Spawn("activate", "ActiveState-CLI/Python"+version)
	suite.Expect("Where would you like to checkout")
	suite.SendLine(tempDir)
	suite.Expect("Downloading")
	suite.Expect("Installing", 120*time.Second)
	suite.Expect("activated state", 120*time.Second)
	suite.WaitForInput()
	suite.SendLine(pythonExe + " -c \"import sys; print(sys.copyright)\"")
	suite.Expect("ActiveState Software Inc.")
	suite.SendLine(pythonExe + " -c \"import numpy; print(numpy.__doc__)\"")
	suite.Expect("import numpy as np")
	suite.SendLine("exit")
	suite.Wait()
}

func TestActivateIntegrationTestSuite(t *testing.T) {
	_ = suite.Run // vscode won't show test helpers unless I use this .. -.-

	//suite.Run(t, new(ActivateIntegrationTestSuite))
	expect.RunParallel(t, new(ActivateIntegrationTestSuite))
}
