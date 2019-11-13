package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/stretchr/testify/suite"
)

type RunIntegrationTestSuite struct {
	integration.Suite
}

func (suite *RunIntegrationTestSuite) prepareTempDirectory(prefix string) (tempDir string, cleanup func()) {

	tempDir, err := ioutil.TempDir("", prefix)
	suite.Require().NoError(err)
	err = os.RemoveAll(tempDir)
	suite.Require().NoError(err)
	err = os.MkdirAll(tempDir, 0770)
	suite.Require().NoError(err)
	suite.Require().NoError(err)
	suite.SetWd(tempDir)

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

	return tempDir, func() {
		os.Chdir(os.TempDir())
		os.RemoveAll(tempDir)
	}
}

func (suite *RunIntegrationTestSuite) BeforeTest(suiteName, testName string) {
	fmt.Printf("Run before test: %s %s", suiteName, testName)
}

func (suite *RunIntegrationTestSuite) TestUninterruptedRun() {

	_, cb := suite.prepareTempDirectory("activate_run_non_interrupt")
	defer cb()
	defer suite.TeardownTest()

	suite.Spawn("run", "test")
	suite.Expect("start of script", 4*time.Second)
	// wait for one second
	time.Sleep(time.Second)
	suite.Expect("ONLY PRINT IF NOT INTERRUPTED", 5*time.Second)
	res, err := suite.Wait(1 * time.Second)
	suite.Require().NoError(err)
	suite.Require().Equal(0, res.ExitCode())
}

func (suite *RunIntegrationTestSuite) TestInterruptedRun() {

	_, cb := suite.prepareTempDirectory("activate_run_interrupt")
	defer cb()
	defer suite.TeardownTest()

	suite.Spawn("run", "test")
	suite.Expect("start of script", 2*time.Second)
	// wait for one second
	time.Sleep(time.Second)
	suite.Send(string([]byte{0x03}))
	suite.Expect("^C", time.Second)
	res, err := suite.Wait(1 * time.Second)
	suite.Require().NotContains(suite.TerminalSnapshot(), "ONLY PRINT IF NOT INTERRUPTED")
	suite.Require().NoError(err)
	suite.Require().Equal(-1, res.ExitCode())
}

func TestRunIntegrationTestSuite(t *testing.T) {
	_ = suite.Run // vscode won't show test helpers unless I use this .. -.-

	//suite.Run(t, new(ActivateIntegrationTestSuite))
	integration.RunParallel(t, new(RunIntegrationTestSuite))
}
