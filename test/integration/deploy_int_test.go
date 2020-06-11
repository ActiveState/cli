package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	switch runtime.GOOS {
	case "windows":
		cp = ts.Spawn("deploy", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work)
	case "darwin":
		// On MacOS the command is the same as Linux, however some binaries
		// already exist at /usr/local/bin so we use the --force flag
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work, "--force"),
			e2e.AppendEnv("SHELL=bash"),
		)
	default:
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work),
			e2e.AppendEnv("SHELL=bash"),
		)
	}

	cp.Expect("Installing", 20*time.Second)
	cp.Expect("Configuring", 20*time.Second)
	cp.Expect("Symlinking", 30*time.Second)
	cp.Expect("Deployment Information", 60*time.Second)
	cp.Expect(ts.Dirs.Work) // expect bin dir
	if runtime.GOOS == "windows" {
		cp.Expect("log out")
	} else {
		cp.Expect("restart")
	}
	cp.ExpectExitCode(0)

	// Linux/Mac symlinks to /usr/local/bin, so we can verify right away
	if runtime.GOOS != "windows" {
		execPath, err := exec.LookPath("python3")
		suite.Require().NoError(err)
		link, err := os.Readlink(execPath)
		suite.Require().NoError(err)
		suite.Contains(link, ts.Dirs.Work, "python3 executable resolves to the one on our target dir")
	}

	cmdIfy := func(args ...string) *termtest.ConsoleProcess {
		if runtime.GOOS != "windows" {
			return ts.SpawnCmd(args[0], args[1:]...)
		}

		var b strings.Builder
		b.WriteString(args[0] + ".lnk")
		for i := 1; i < len(args); i++ {
			b.WriteString(fmt.Sprintf(" %s", args[i]))
		}

		return ts.SpawnCmd("cmd", "/c", b.String())
	}

	// check that some of the installed symlinks are use-able
	cp = cmdIfy(filepath.Join(ts.Dirs.Work, "bin", "python3"), "--version")

	cp.Expect("Python 3")
	cp.ExpectExitCode(0)

	cp = cmdIfy(filepath.Join(ts.Dirs.Work, "bin", "pip3"), "--version")
	cp.Expect("pip")
	cp.ExpectExitCode(0)

	if runtime.GOOS == "darwin" {
		// This is kept as a regression test, pyvenv used to have a relocation problem on MacOS
		cp = cmdIfy(filepath.Join(ts.Dirs.Work, "bin", "pyvenv"), "-h")
		cp.ExpectExitCode(0)
	}

	cp = cmdIfy(filepath.Join(ts.Dirs.Work, "bin", "python3"), "-m", "pytest", "--version")
	cp.Expect("This is pytest version")

	if runtime.GOOS != "windows" {
		// AzureCI has multiple representations for the work directory that
		// may not agree when running tests
		cp.Expect(fmt.Sprintf("imported from %s", ts.Dirs.Work))
	}

	cp.ExpectExitCode(0)

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

	if runtime.GOOS == "windows" {
		cp = ts.Spawn("deploy", "configure", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work, "--user")
		cp.Expect("Configuring shell", 60*time.Second)
		cp.ExpectExitCode(0)

		out, err := exec.Command("reg", "query", `HKCU\Environment`, "/v", "Path").Output()
		suite.Require().NoError(err)
		suite.Contains(string(out), ts.Dirs.Work, "Windows user PATH should contain our target dir")
	}
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
		out, err := exec.Command("reg", "query", `HKLM\SYSTEM\ControlSet001\Control\Session Manager\Environment`, "/v", "Path").Output()
		suite.Require().NoError(err)
		suite.Contains(string(out), ts.Dirs.Work, "Windows system PATH should contain our target dir")
	}
}

func (suite *DeployIntegrationTestSuite) TestDeploySymlink() {
	if runtime.GOOS != "windows" && !e2e.RunningOnCI() {
		suite.T().Skipf("Skipping TestDeploySymlink when not running on CI, as it modifies PATH")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Install step is required
	cp := ts.Spawn("deploy", "symlink", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work)
	cp.Expect("need to run the install step")
	cp.ExpectExitCode(1)
	suite.InstallAndAssert(ts)

	pathDir := fileutils.TempDirUnsafe()
	if runtime.GOOS != "darwin" {
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", "symlink", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work),
			e2e.AppendEnv(fmt.Sprintf("PATH=%s", pathDir)), // Avoid conflicts
		)
	} else {
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", "symlink", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work, "--force"),
		)
	}

	cp.Expect("Symlinking executables")
	cp.ExpectExitCode(0)

	if runtime.GOOS == "linux" {
		suite.True(fileutils.FileExists(filepath.Join(pathDir, "python3"+symlinkExt)), "Python3 symlink should have been written to PATH")
	}
	suite.True(fileutils.FileExists(filepath.Join(ts.Dirs.Work, "bin", "python3"+symlinkExt)), "Python3 symlink should have been written")
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

func (suite *DeployIntegrationTestSuite) TestDeployTwice() {
	if runtime.GOOS == "darwin" || !e2e.RunningOnCI() {
		suite.T().Skipf("Skipping TestDeployTwice when not running on CI or on MacOS, as it modifies PATH")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.InstallAndAssert(ts)

	pathDir := fileutils.TempDirUnsafe()
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("deploy", "symlink", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work),
		e2e.AppendEnv(fmt.Sprintf("PATH=%s", pathDir)), // Avoid conflicts
	)
	cp.ExpectExitCode(0)

	suite.True(fileutils.FileExists(filepath.Join(ts.Dirs.Work, "bin", "python3"+symlinkExt)), "Python3 symlink should have been written")

	// Running deploy a second time should not cause any errors (cache is properly picked up)
	cpx := ts.SpawnWithOpts(
		e2e.WithArgs("deploy", "symlink", "ActiveState-CLI/Python3", "--path", ts.Dirs.Work),
		e2e.AppendEnv(fmt.Sprintf("PATH=%s", pathDir)), // Avoid conflicts
	)
	cpx.ExpectExitCode(0)
}

func TestDeployIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DeployIntegrationTestSuite))
}
