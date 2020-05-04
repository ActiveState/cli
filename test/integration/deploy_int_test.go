package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
)

type DeployIntegrationTestSuite struct {
	suite.Suite
}

var symlinkExt = ""

func init() {
	if runtime.GOOS == "windows" {
		symlinkExt = ".lnk"
	}
}

func (suite *DeployIntegrationTestSuite) TestDeploy() {
	if !e2e.RunningOnCI() {
		suite.T().Skipf("Skipping DeployIntegrationTestSuite when not running on CI, as it modifies bashrc/registry")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	var cp *termtest.ConsoleProcess
	if runtime.GOOS != "windows" {
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work),
			e2e.AppendEnv("SHELL=bash"),
		)
	} else {
		cp = ts.Spawn("deploy", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work, "--force")
	}

	cp.Expect("Installing", 120*time.Second)
	cp.Expect("Configuring", 120*time.Second)
	cp.Expect("Symlinking")
	cp.Expect("Deployment Information", 60*time.Second)
	cp.Expect(ts.Dirs.Work) // expect bin dir
	if runtime.GOOS == "windows" {
		cp.Expect("log out")
	} else {
		cp.Expect("restart")
	}
	cp.ExpectExitCode(0)

	// Linux symlinks to /usr/local/bin, so we can verify right away
	if runtime.GOOS == "linux" {
		execPath, err := exec.LookPath("python3")
		suite.Require().NoError(err)
		link, err := os.Readlink(execPath)
		suite.Require().NoError(err)
		suite.Contains(link, ts.Dirs.Work, "python3 executable resolves to the one on our target dir")
	}

	suite.AssertConfig(ts)
}

func (suite *DeployIntegrationTestSuite) TestDeployInstall() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	isEmpty, fail := fileutils.IsEmptyDir(ts.Dirs.Work)
	suite.Require().NoError(fail.ToError())
	suite.True(isEmpty, "Target dir should be empty before we start")

	suite.InstallAndAssert(ts)

	isEmpty, fail = fileutils.IsEmptyDir(ts.Dirs.Work)
	suite.Require().NoError(fail.ToError())
	suite.False(isEmpty, "Target dir should have artifacts written to it")
}

func (suite *DeployIntegrationTestSuite) InstallAndAssert(ts *e2e.Session) {
	cp := ts.Spawn("deploy", "install", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work)

	cp.Expect("Installing Runtime")
	cp.Expect("Downloading")
	cp.Expect("Installing", 120*time.Second)
	cp.Expect("Installation completed", 120*time.Second)
	cp.ExpectExitCode(0)
}

func (suite *DeployIntegrationTestSuite) TestDeployConfigure() {
	if !e2e.RunningOnCI() {
		suite.T().Skipf("Skipping TestDeployConfigure when not ru" +
			"nning on CI, as it modifies bashrc/registry")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Install step is required
	cp := ts.Spawn("deploy", "configure", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work)
	cp.Expect("need to run the install step")
	cp.ExpectExitCode(1)
	suite.InstallAndAssert(ts)

	if runtime.GOOS != "windows" {
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", "configure", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work),
			e2e.AppendEnv("SHELL=bash"),
		)
	} else {
		cp = ts.Spawn("deploy", "configure", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work)
	}

	cp.Expect("Configuring shell", 60*time.Second)
	cp.ExpectExitCode(0)

	suite.AssertConfig(ts)
}

func (suite *DeployIntegrationTestSuite) AssertConfig(ts *e2e.Session) {
	if runtime.GOOS != "windows" {
		// Test bashrc
		homeDir, err := os.UserHomeDir()
		suite.Require().NoError(err)

		bashContents := fileutils.ReadFileUnsafe(filepath.Join(homeDir, ".bashrc"))
		suite.Contains(string(bashContents), constants.RCAppendStartLine, "bashrc should contain our RC Append Start line")
		suite.Contains(string(bashContents), constants.RCAppendStopLine, "bashrc should contain our RC Append Stop line")
		suite.Contains(string(bashContents), ts.Dirs.Work, "bashrc should contain our target dir")
	} else {
		// Test registry
		out, err := exec.Command("reg", "query", "HKCU\\Environment", "/v", "Path").Output()
		suite.Require().NoError(err)
		suite.Contains(out, ts.Dirs.Work, "Windows PATH should contain our target dir")
	}
}

func (suite *DeployIntegrationTestSuite) TestDeploySymlink() {
	if runtime.GOOS == "linux" && !e2e.RunningOnCI() {
		suite.T().Skipf("Skipping TestDeploySymlink when not running on CI, as it modifies PATH")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Install step is required
	cp := ts.Spawn("deploy", "symlink", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work)
	cp.Expect("need to run the install step")
	cp.ExpectExitCode(1)
	suite.InstallAndAssert(ts)

	cp = ts.Spawn("deploy", "symlink", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work, "--force")

	cp.Expect("Symlinking executables")
	cp.ExpectExitCode(0)

	suite.True(fileutils.FileExists(filepath.Join(ts.Dirs.Work, "bin", "python3"+symlinkExt)), "Python3 symlink should have been written")

	// Linux symlinks to /usr/local/bin, so we can verify right away
	if runtime.GOOS == "linux" {
		execPath, err := exec.LookPath("python3")
		suite.Require().NoError(err)
		link, err := os.Readlink(execPath)
		suite.Require().NoError(err)
		suite.Contains(link, ts.Dirs.Work, "python3 executable resolves to the one on our target dir")
	}
}

func (suite *DeployIntegrationTestSuite) TestDeployReport() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Install step is required
	cp := ts.Spawn("deploy", "report", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work)
	cp.Expect("need to run the install step")
	cp.ExpectExitCode(1)
	suite.InstallAndAssert(ts)

	cp = ts.Spawn("deploy", "report", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work)
	cp.Expect("Deployment Information")
	cp.Expect(ts.Dirs.Work) // expect bin dir
	if runtime.GOOS == "windows" {
		cp.Expect("log out")
	} else {
		cp.Expect("restart")
	}
	cp.ExpectExitCode(0)
}

func TestDeployIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DeployIntegrationTestSuite))
}
