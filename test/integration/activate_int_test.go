package integration

import (
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

func (suite *ActivateIntegrationTestSuite) prepareTempDirectory(prefix string) (tempDir string, cleanup func()) {

	tempDir, err := ioutil.TempDir("", prefix)
	suite.Require().NoError(err)
	err = os.RemoveAll(tempDir)
	suite.Require().NoError(err)
	err = os.MkdirAll(tempDir, 0770)
	suite.Require().NoError(err)
	err = os.Chdir(tempDir)
	suite.Require().NoError(err)

	return tempDir, func() {
		os.Chdir(os.TempDir())
		os.RemoveAll(tempDir)
	}
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython3() {
	suite.activatePython("3")
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython2() {
	suite.T().Skip("Python 2 is not officially supported by the platform ATM.")
	suite.activatePython("2")
}

func (suite *ActivateIntegrationTestSuite) TestActivateWithoutRuntime() {

	tempDir, cb := suite.prepareTempDirectory("activate_test_no_runtime")
	defer cb()

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
	if runtime.GOOS == "darwin" {
		suite.T().Skip("Runtimes are not supported on macOS")
	}
	if runtime.GOOS == "windows" {
		suite.T().Skip("suite.AppendEnv() does not work on windows currently.  Skipping this test.")
	}

	pythonExe := "python" + version

	tempDir, cb := suite.prepareTempDirectory("activate_test")
	defer cb()

	suite.LoginAsPersistentUser()
	suite.AppendEnv([]string{"ACTIVESTATE_CLI_DISABLE_RUNTIME=false"})

	suite.Spawn("activate", "ActiveState-CLI/Python"+version)
	suite.Expect("Where would you like to checkout")
	suite.SendLine(tempDir)
	suite.Expect("Downloading", 120*time.Second)
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
