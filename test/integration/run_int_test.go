package integration

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
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

	root := environment.GetRootPathUnsafe()
	interruptScript := filepath.Join(root, "test", "integration", "assets", "run", "interrupt.sh")
	fileutils.CopyFile(interruptScript, filepath.Join(projectDir, "interrupt.sh"))

	configFileContent := strings.TrimSpace(`
project: https://platform.activestate.com/Owner/ProjectName
scripts:
  - name: test
    description: A script that runs for 20 seconds doing nothing.  It should be interrupted.
    standalone: true
    value: bash interrupt.sh
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
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("bash"); err != nil {
			suite.T().Skip("This test requires a bash shell in your PATH")
		}
	}
	tmpDir, cleanup := suite.PrepareTemporaryWorkingDirectory(suite.T().Name())
	suite.tmpDirCleanup = cleanup

	suite.createProjectFile(tmpDir)
}

func (suite *RunIntegrationTestSuite) TearDownTest() {
	suite.Suite.TearDownTest()
	suite.tmpDirCleanup()
}

func (suite *RunIntegrationTestSuite) TestOneInterrupt() {

	suite.Spawn("run", "test")
	suite.Expect("Start of script")
	time.Sleep(500 * time.Millisecond)
	// interrupt the first (very long sleep)
	suite.SendCtrlC()

	suite.Expect("received SIGINT", 3*time.Second)
	suite.Expect("After first sleep or interrupt", 2*time.Second)
	suite.Expect("After second sleep")
	res, err := suite.Wait(20 * time.Second)
	suite.Require().NoError(err)
	suite.Require().Equal(0, res.ExitCode())
}

func (suite *RunIntegrationTestSuite) TestTwoInterrupts() {
	suite.Spawn("run", "test")
	suite.Expect("Start of script")
	time.Sleep(500 * time.Millisecond)
	suite.SendCtrlC()
	suite.Expect("received SIGINT", 3*time.Second)
	suite.Expect("After first sleep or interrupt", 2*time.Second)
	time.Sleep(500 * time.Millisecond)
	suite.SendCtrlC()
	res, err := suite.Wait(20 * time.Second)
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
