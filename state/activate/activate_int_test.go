package activate_test

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

func (suite *ActivateIntegrationTestSuite) TestActivatePython3() {
	suite.activatePython("3")
}

func (suite *ActivateIntegrationTestSuite) TestActivatePython2() {
	suite.activatePython("2")
}

func (suite *ActivateIntegrationTestSuite) TestActivateWithoutRuntime() {
	os.Chdir(os.TempDir())

	tempDir, err := ioutil.TempDir("", "")
	suite.Require().NoError(err)
	os.Remove(tempDir)
	suite.Require().NoError(err)

	suite.LoginAsPersistentUser()

	suite.Spawn("activate", "ActiveState-CLI/Python3")
	suite.Expect("Where would you like to checkout")
	suite.Send(tempDir)
	suite.Expect("State activated") // Note this line is REQUIRED. For reasons I cannot figure out the below WaitForInput will fail unless the subshell prints something.
	suite.WaitForInput(10 * time.Second)
	suite.Send("exit")
	suite.Wait()
}

func (suite *ActivateIntegrationTestSuite) activatePython(version string) {
	if runtime.GOOS == "darwin" {
		return // Runtimes aren't supported on macOS
	}

	pythonExe := "python" + version

	os.Chdir(os.TempDir())

	suite.LoginAsPersistentUser()
	suite.AppendEnv([]string{"ACTIVESTATE_CLI_DISABLE_RUNTIME=false"})

	tempDir, err := ioutil.TempDir("", "")
	suite.Require().NoError(err)
	os.Remove(tempDir)
	suite.Require().NoError(err)

	suite.Spawn("activate", "ActiveState-CLI/Python"+version)
	suite.Expect("Where would you like to checkout")
	suite.Send(tempDir)
	suite.Expect("Downloading")
	suite.Expect("Installing", 120*time.Second)
	suite.Expect("State activated")
	suite.WaitForInput(120 * time.Second)
	suite.Send(pythonExe + " -c \"import sys; print(sys.copyright)\"")
	suite.Expect("ActiveState Software Inc.")
	suite.Send(pythonExe + " -c \"import numpy; print(numpy.__doc__)\"")
	suite.Expect("import numpy as np")
	suite.Send("exit")
	suite.Wait()
}

func TestActivateIntegrationTestSuite(t *testing.T) {
	_ = suite.Run // vscode won't show test helpers unless I use this .. -.-

	//suite.Run(t, new(ActivateIntegrationTestSuite))
	expect.RunParallel(t, new(ActivateIntegrationTestSuite))
}
