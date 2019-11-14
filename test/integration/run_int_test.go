package integration

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

type RunIntegrationTestSuite struct {
	integration.Suite
	tmpDirCleanup func()
}

func (suite *RunIntegrationTestSuite) createProjectFile(projectDir string) {
	configFileContent := strings.TrimSpace(`
project: https://platform.activestate.com/Owner/ProjectName
scripts:
  - name: test
    description: A script that runs for 20 seconds doing nothing.  It should be interrupted.
    standalone: true
    value: |
       bash -c "
            function f() { echo received SIGINT; }
            trap f SIGINT
            trap -p SIGINT
            echo 'Start of script'
            sleep 10000          
            echo 'After first sleep or interrupt'
            trap 'exit 123' SIGINT
            sleep 2
            echo 'After second sleep, but not printed after second interrupt'
          "
`)
	projectFile := &projectfile.Project{}
	err := yaml.Unmarshal([]byte(configFileContent), projectFile)
	suite.Require().NoError(err)

	projectFile.SetPath(filepath.Join(projectDir, constants.ConfigFileName))
	fail := projectFile.Save()
	suite.Require().NoError(fail.ToError())

}

func (suite *RunIntegrationTestSuite) SetupTest() {
	suite.Suite.SetupTest()
	tmpDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	suite.tmpDirCleanup = cleanup

	suite.createProjectFile(tmpDir)
}

func (suite *RunIntegrationTestSuite) TeardownTest() {
	suite.Suite.TeardownTest()
	suite.tmpDirCleanup()
}

func (suite *RunIntegrationTestSuite) TestOneInterrupt() {

	suite.Spawn("run", "test")
	suite.Expect("Start of script")
	time.Sleep(200 * time.Millisecond)
	// interrupt the first (very long sleep)
	suite.Send(string([]byte{3}))
	suite.SendCtrlC()

	suite.Expect("received SIGINT", 3*time.Second)
	suite.Expect("After first sleep or interrupt", 2*time.Second)
	suite.Expect("After second sleep")
	res, err := suite.Wait(1 * time.Second)
	suite.Require().NoError(err)
	suite.Require().Equal(0, res.ExitCode())
}

func (suite *RunIntegrationTestSuite) TestTwoInterrupts() {
	suite.Spawn("run", "test")
	suite.Expect("Start of script")
	time.Sleep(200 * time.Millisecond)
	suite.SendCtrlC()
	suite.Expect("received SIGINT", 3*time.Second)
	suite.Expect("After first sleep or interrupt", 2*time.Second)
	time.Sleep(200 * time.Millisecond)
	suite.SendCtrlC()
	res, err := suite.Wait(3 * time.Second)
	suite.Require().NoError(err)
	suite.Require().Equal(123, res.ExitCode())
	suite.Require().NotContains(
		suite.TerminalSnapshot(), "not printed after second interrupt",
	)
}

func TestRunIntegrationTestSuite(t *testing.T) {
	_ = suite.Run // vscode won't show test helpers unless I use this .. -.-

	//suite.Run(t, new(ActivateIntegrationTestSuite))
	integration.RunParallel(t, new(RunIntegrationTestSuite))
}
