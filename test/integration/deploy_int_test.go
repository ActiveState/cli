package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ActiveState/termtest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type DeployIntegrationTestSuite struct {
	tagsuite.Suite
}

var symlinkExt = ""

func init() {
	if runtime.GOOS == "windows" {
		symlinkExt = ".lnk"
	}
}

func (suite *DeployIntegrationTestSuite) deploy(ts *e2e.Session, prj string, targetPath string, targetID string) {
	var cp *termtest.ConsoleProcess
	switch runtime.GOOS {
	case "windows":
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", prj, "--path", targetPath),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		)
	case "darwin":
		// On MacOS the command is the same as Linux, however some binaries
		// already exist at /usr/local/bin so we use the --force flag
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", prj, "--path", targetPath, "--force"),
			e2e.AppendEnv("SHELL=bash"),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		)
	default:
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", prj, "--path", targetPath),
			e2e.AppendEnv("SHELL=bash"),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		)
	}

	cp.Expect("Installing", 40*time.Second)
	cp.Expect("Configuring", 40*time.Second)
	if runtime.GOOS != "windows" {
		cp.Expect("Symlinking", 30*time.Second)
	}
	cp.Expect("Deployment Information", 60*time.Second)
	cp.Expect(targetID) // expect bin dir
	if runtime.GOOS == "windows" {
		cp.Expect("log out")
	} else {
		cp.Expect("restart")
	}
	cp.ExpectExitCode(0)
}

func (suite *DeployIntegrationTestSuite) TestDeployPerl() {
	suite.OnlyRunForTags(tagsuite.Perl, tagsuite.Deploy)
	if !e2e.RunningOnCI() {
		suite.T().Skipf("Skipping DeployIntegrationTestSuite when not running on CI, as it modifies bashrc/registry")
	}

	if runtime.GOOS == "darwin" {
		suite.T().Skip("Perl is not supported on Mac OS yet.")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	targetID, err := uuid.NewUUID()
	suite.Require().NoError(err)
	targetPath, err := fileutils.ResolveUniquePath(filepath.Join(ts.Dirs.Work, targetID.String()))
	suite.Require().NoError(err)

	suite.deploy(ts, "ActiveState-CLI/Perl", targetPath, targetID.String())

	suite.checkSymlink("perl", ts.Dirs.Bin, targetID.String())

	var cp *termtest.ConsoleProcess
	if runtime.GOOS == "windows" {
		cp = ts.SpawnCmdWithOpts(
			"cmd.exe",
			e2e.WithArgs("/k", filepath.Join(targetPath, "bin", "shell.bat")),
			e2e.AppendEnv("PATHEXT=.COM;.EXE;.BAT;.LNK", "SHELL="),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		)
	} else {
		cp = ts.SpawnCmdWithOpts(
			"/bin/bash",
			e2e.AppendEnv("PROMPT_COMMAND="),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"))
		cp.SendLine(fmt.Sprintf("source %s\n", filepath.Join(targetPath, "bin", "shell.sh")))
	}

	errorLevel := "echo $?"
	if runtime.GOOS == "windows" {
		errorLevel = `echo %ERRORLEVEL%`
	}
	// check that some of the installed binaries are use-able
	cp.SendLine("perl --version")
	cp.Expect("This is perl 5")
	cp.SendLine(errorLevel)
	cp.Expect("0")

	cp.SendLine("ptar -h")
	cp.Expect("a tar-like program written in perl")

	cp.SendLine("exit 0")
	cp.ExpectExitCode(0)
}

func (suite *DeployIntegrationTestSuite) checkSymlink(name string, binDir, targetID string) {
	if runtime.GOOS != "Linux" {
		return
	}
	// Linux symlinks to /usr/local/bin or the first write-able directory in PATH, so we can verify right away
	execPath, err := exec.LookPath(name)
	// If not on PATH it needs to exist in the temporary directory
	var execDir string
	if err == nil {
		execDir, _ = filepath.Split(execPath)
	}
	if err != nil || (execDir != "/usr/local/bin/" && execDir != "/usr/bin/") {
		execPath = filepath.Join(binDir, name)
		if !fileutils.FileExists(execPath) {
			suite.Fail("Expected to find %s on PATH", name)
		}
	}
	link, err := os.Readlink(execPath)
	suite.Require().NoError(err)
	suite.Contains(link, targetID, "%s executable resolves to the one on our target dir", name)
}

func (suite *DeployIntegrationTestSuite) TestDeployPython() {
	suite.OnlyRunForTags(tagsuite.Deploy, tagsuite.Python, tagsuite.Critical)
	if !e2e.RunningOnCI() {
		suite.T().Skipf("Skipping DeployIntegrationTestSuite when not running on CI, as it modifies bashrc/registry")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	targetID, err := uuid.NewUUID()
	suite.Require().NoError(err)
	targetPath, err := fileutils.ResolveUniquePath(filepath.Join(ts.Dirs.Work, targetID.String()))
	suite.Require().NoError(err)

	suite.deploy(ts, "ActiveState-CLI/Python3", targetPath, targetID.String())

	suite.checkSymlink("python3", ts.Dirs.Bin, targetID.String())

	var cp *termtest.ConsoleProcess
	if runtime.GOOS == "windows" {
		cp = ts.SpawnCmdWithOpts(
			"cmd.exe",
			e2e.WithArgs("/k", filepath.Join(targetPath, "bin", "shell.bat")),
			e2e.AppendEnv("PATHEXT=.COM;.EXE;.BAT;.LNK", "SHELL="),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		)
	} else {
		cp = ts.SpawnCmdWithOpts(
			"/bin/bash",
			e2e.AppendEnv("PROMPT_COMMAND="),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"))
		cp.SendLine(fmt.Sprintf("source %s\n", filepath.Join(targetPath, "bin", "shell.sh")))
	}

	errorLevel := "echo $?"
	if runtime.GOOS == "windows" {
		errorLevel = `echo %ERRORLEVEL%`
	}

	cp.SendLine("python3 --version")
	cp.Expect("Python 3")
	cp.SendLine(errorLevel)
	cp.Expect("0")

	cp.SendLine("pip3 --version")
	cp.Expect("pip")
	cp.SendLine(errorLevel)
	cp.Expect("0")

	if runtime.GOOS == "darwin" {
		// This is kept as a regression test, pyvenv used to have a relocation problem on MacOS
		cp.SendLine("pyvenv -h")
		cp.SendLine("echo $?")
		cp.Expect("0")
	}

	cp.SendLine("python3 -m pytest --version")
	cp.Expect("pytest")

	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	suite.AssertConfig(ts, targetID.String())
}

func (suite *DeployIntegrationTestSuite) TestDeployInstall() {
	suite.OnlyRunForTags(tagsuite.Deploy)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	targetDir, err := fileutils.ResolveUniquePath(filepath.Join(ts.Dirs.Work, "target"))
	suite.Require().NoError(err)
	if fileutils.TargetExists(targetDir) {
		isEmpty, err := fileutils.IsEmptyDir(targetDir)
		suite.Require().NoError(err)
		suite.True(isEmpty, "Target dir should be empty before we start")
	}

	suite.InstallAndAssert(ts, targetDir)

	isEmpty, err := fileutils.IsEmptyDir(targetDir)
	suite.Require().NoError(err)
	suite.False(isEmpty, "Target dir should have artifacts written to it")
}

func (suite *DeployIntegrationTestSuite) InstallAndAssert(ts *e2e.Session, targetPath string) {
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("deploy", "install", "ActiveState-CLI/Python3", "--path", targetPath),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)

	cp.Expect("Installing Runtime")
	cp.Expect("Installing", 120*time.Second)
	cp.Expect("Installation completed", 120*time.Second)
	cp.ExpectExitCode(0)
}

func (suite *DeployIntegrationTestSuite) TestDeployConfigure() {
	suite.OnlyRunForTags(tagsuite.Deploy)
	if !e2e.RunningOnCI() {
		suite.T().Skipf("Skipping TestDeployConfigure when not running on CI, as it modifies bashrc/registry")
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	targetID, err := uuid.NewUUID()
	suite.Require().NoError(err)
	targetPath, err := fileutils.ResolveUniquePath(filepath.Join(ts.Dirs.Work, targetID.String()))
	suite.Require().NoError(err)

	// Install step is required
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("deploy", "configure", "ActiveState-CLI/Python3", "--path", targetPath),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("need to run the install step")
	cp.ExpectExitCode(1)
	suite.InstallAndAssert(ts, targetPath)

	if runtime.GOOS != "windows" {
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", "configure", "ActiveState-CLI/Python3", "--path", targetPath),
			e2e.AppendEnv("SHELL=bash"),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		)
	} else {
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", "configure", "ActiveState-CLI/Python3", "--path", targetPath),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		)
	}

	cp.Expect("Configuring shell", 60*time.Second)
	cp.ExpectExitCode(0)
	suite.AssertConfig(ts, targetID.String())

	if runtime.GOOS == "windows" {
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", "configure", "ActiveState-CLI/Python3", "--path", targetPath, "--user"),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		)
		cp.Expect("Configuring shell", 60*time.Second)
		cp.ExpectExitCode(0)

		out, err := exec.Command("reg", "query", `HKCU\Environment`, "/v", "Path").Output()
		suite.Require().NoError(err)
		suite.Contains(string(out), targetID.String(), "Windows user PATH should contain our target dir")
	}
}

func (suite *DeployIntegrationTestSuite) AssertConfig(ts *e2e.Session, targetID string) {
	if runtime.GOOS != "windows" {
		// Test bashrc
		homeDir, err := os.UserHomeDir()
		suite.Require().NoError(err)

		cfg, err := config.New()
		suite.Require().NoError(err)

		subshell := subshell.New(cfg)
		rcFile, err := subshell.RcFile()
		suite.Require().NoError(err)

		bashContents := fileutils.ReadFileUnsafe(filepath.Join(homeDir, rcFile))
		suite.Contains(string(bashContents), constants.RCAppendDeployStartLine, "bashrc should contain our RC Append Start line")
		suite.Contains(string(bashContents), constants.RCAppendDeployStopLine, "bashrc should contain our RC Append Stop line")
		suite.Contains(string(bashContents), targetID, "bashrc should contain our target dir")
	} else {
		// Test registry
		out, err := exec.Command("reg", "query", `HKLM\SYSTEM\ControlSet001\Control\Session Manager\Environment`, "/v", "Path").Output()
		suite.Require().NoError(err)
		suite.Contains(string(out), targetID, "bashrc should contain our target dir")
	}
}

func (suite *DeployIntegrationTestSuite) TestDeploySymlink() {
	suite.OnlyRunForTags(tagsuite.Deploy)
	if runtime.GOOS != "windows" && !e2e.RunningOnCI() {
		suite.T().Skipf("Skipping TestDeploySymlink when not running on CI, as it modifies PATH")
	}

	ts := e2e.New(suite.T(), false, "SHELL=")
	defer ts.Close()

	targetID, err := uuid.NewUUID()
	suite.Require().NoError(err)
	targetPath, err := fileutils.ResolveUniquePath(filepath.Join(ts.Dirs.Work, targetID.String()))
	suite.Require().NoError(err)

	// Install step is required
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("deploy", "symlink", "ActiveState-CLI/Python3", "--path", targetPath),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("need to run the install step")
	cp.ExpectExitCode(1)
	suite.InstallAndAssert(ts, targetPath)

	if runtime.GOOS != "darwin" {
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", "symlink", "ActiveState-CLI/Python3", "--path", targetPath),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		)
	} else {
		cp = ts.SpawnWithOpts(
			e2e.WithArgs("deploy", "symlink", "ActiveState-CLI/Python3", "--path", targetPath, "--force"),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		)
	}

	if runtime.GOOS != "windows" {
		cp.Expect("Symlinking executables")
	}
	cp.ExpectExitCode(0)

	suite.checkSymlink("python3", ts.Dirs.Bin, targetID.String())
}

func (suite *DeployIntegrationTestSuite) TestDeployReport() {
	suite.OnlyRunForTags(tagsuite.Deploy)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	targetID, err := uuid.NewUUID()
	suite.Require().NoError(err)
	targetPath, err := fileutils.ResolveUniquePath(filepath.Join(ts.Dirs.Work, targetID.String()))
	suite.Require().NoError(err)

	// Install step is required
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("deploy", "report", "ActiveState-CLI/Python3", "--path", targetPath),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("need to run the install step")
	cp.ExpectExitCode(1)
	suite.InstallAndAssert(ts, targetPath)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("deploy", "report", "ActiveState-CLI/Python3", "--path", targetPath),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Deployment Information")
	cp.Expect(targetID.String()) // expect bin dir
	if runtime.GOOS == "windows" {
		cp.Expect("log out")
	} else {
		cp.Expect("restart")
	}
	cp.ExpectExitCode(0)
}

func (suite *DeployIntegrationTestSuite) TestDeployTwice() {
	suite.OnlyRunForTags(tagsuite.Deploy)
	if runtime.GOOS == "darwin" || !e2e.RunningOnCI() {
		suite.T().Skipf("Skipping TestDeployTwice when not running on CI or on MacOS, as it modifies PATH")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	targetPath, err := fileutils.ResolveUniquePath(filepath.Join(ts.Dirs.Work, "target"))
	suite.Require().NoError(err)

	suite.InstallAndAssert(ts, targetPath)

	pathDir := fileutils.TempDirUnsafe()
	defer os.RemoveAll(pathDir)
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("deploy", "symlink", "ActiveState-CLI/Python3", "--path", targetPath),
		e2e.AppendEnv(fmt.Sprintf("PATH=%s", pathDir)), // Avoid conflicts
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.ExpectExitCode(0)

	// we do not symlink on windows anymore
	if runtime.GOOS != "windows" {
		suite.True(fileutils.FileExists(filepath.Join(targetPath, "bin", "python3"+symlinkExt)), "Python3 symlink should have been written")
	}

	// Running deploy a second time should not cause any errors (cache is properly picked up)
	cpx := ts.SpawnWithOpts(
		e2e.WithArgs("deploy", "symlink", "ActiveState-CLI/Python3", "--path", targetPath),
		e2e.AppendEnv(fmt.Sprintf("PATH=%s", pathDir)), // Avoid conflicts
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cpx.ExpectExitCode(0)
}

func TestDeployIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DeployIntegrationTestSuite))
}
