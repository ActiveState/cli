package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/termtest"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/bash"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
	"github.com/ActiveState/cli/internal/subshell/zsh"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ShellIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ShellIntegrationTestSuite) TestShell() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/small-python")
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	args := []string{"small-python", "ActiveState-CLI/small-python"}
	for _, arg := range args {
		cp := ts.Spawn("shell", arg)
		cp.Expect("Activated")
		cp.ExpectInput()

		cp.SendLine("python3 --version")
		cp.Expect("Python 3")
		cp.SendLine("exit")
		cp.Expect("Deactivated")
		cp.ExpectExitCode(0)
	}

	// Both Windows and MacOS can run into path comparison issues with symlinks and long paths.
	projectName := "small-python"
	if runtime.GOOS == "linux" {
		projectDir := filepath.Join(ts.Dirs.Work, projectName)
		// projectDir, err := fileutils.SymlinkTarget(projectDir)
		// suite.Require().NoError(err)
		err := os.RemoveAll(projectDir)
		suite.Require().NoError(err)

		cp = ts.Spawn("shell", projectName)
		cp.Expect(fmt.Sprintf("Could not load project %s from path: %s", projectName, projectDir))
	}

	// Check for project not checked out.
	args = []string{"Python-3.9", "ActiveState-CLI/Python-3.9"}
	for _, arg := range args {
		cp := ts.Spawn("shell", arg)
		cp.Expect("Cannot find the Python-3.9 project")
		cp.ExpectExitCode(1)
	}
}

func (suite *ShellIntegrationTestSuite) TestDefaultShell() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Checkout.
	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	// Use.
	cp = ts.Spawn("use", "ActiveState-CLI/Empty")
	cp.Expect("Switched to project")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("shell")
	cp.Expect("Activated")
	cp.ExpectInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ShellIntegrationTestSuite) TestCwdShell() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("activate", "ActiveState-CLI/Empty")
	cp.Expect("Activated")
	cp.ExpectInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("shell"),
		e2e.OptWD(filepath.Join(ts.Dirs.Work, "Empty")),
	)
	cp.Expect("Activated")
	cp.ExpectInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ShellIntegrationTestSuite) TestCd() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("activate", "ActiveState-CLI/Empty")
	cp.Expect("Activated")
	cp.ExpectInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	subdir := filepath.Join(ts.Dirs.Work, "foo", "bar", "baz")
	err := fileutils.Mkdir(subdir)
	suite.Require().NoError(err)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("shell", "ActiveState-CLI/Empty"),
		e2e.OptWD(subdir),
	)
	cp.Expect("Activated")
	cp.ExpectInput()
	if runtime.GOOS != "windows" {
		cp.SendLine("pwd")
	} else {
		cp.SendLine("echo %cd%")
	}
	cp.Expect(subdir)
	cp.SendLine("exit")

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("shell", "ActiveState-CLI/Empty", "--cd"),
		e2e.OptWD(subdir),
	)
	cp.Expect("Activated")
	cp.ExpectInput()
	if runtime.GOOS != "windows" {
		cp.SendLine("ls")
	} else {
		cp.SendLine("dir")
	}
	cp.Expect("activestate.yaml")
	cp.SendLine("exit")

	cp.ExpectExitCode(0)
}

func (suite *ShellIntegrationTestSuite) TestDefaultNoLongerExists() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("use", "ActiveState-CLI/Empty")
	cp.Expect("Switched to project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	err := os.RemoveAll(filepath.Join(ts.Dirs.Work, "Empty"))
	suite.Require().NoError(err)

	cp = ts.Spawn("shell")
	cp.Expect("Cannot find your project")
	cp.ExpectExitCode(1)
}

func (suite *ShellIntegrationTestSuite) TestUseShellUpdates() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.SetupRCFile(ts)

	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	// Create a zsh RC file
	var zshRcFile string
	var err error
	if runtime.GOOS != "windows" {
		zsh := &zsh.SubShell{}
		zshRcFile, err = zsh.RcFile()
		suite.NoError(err)
	}

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("use", "ActiveState-CLI/Empty"),
		e2e.OptAppendEnv("SHELL=bash"),
	)
	cp.Expect("Switched to project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	// Ensure both bash and zsh RC files are updated
	cfg, err := config.New()
	suite.NoError(err)
	rcfile, err := subshell.New(cfg).RcFile()
	if runtime.GOOS != "windows" && fileutils.FileExists(rcfile) {
		suite.NoError(err)
		suite.Contains(string(fileutils.ReadFileUnsafe(rcfile)), ts.Dirs.DefaultBin, "PATH does not have your project in it")
		suite.Contains(string(fileutils.ReadFileUnsafe(zshRcFile)), ts.Dirs.DefaultBin, "PATH does not have your project in it")
	}
}

func (suite *ShellIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Shell, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("shell", "--output", "json")
	cp.Expect(`"error":"This command does not support the 'json' output format`, termtest.OptExpectTimeout(5*time.Second))
	cp.ExpectExitCode(1)
	AssertValidJSON(suite.T(), cp)
}

func (suite *ShellIntegrationTestSuite) SetupRCFile(ts *e2e.Session) {
	if runtime.GOOS == "windows" {
		return
	}

	ts.SetupRCFile()
	ts.SetupRCFileCustom(&zsh.SubShell{})
}

func (suite *ShellIntegrationTestSuite) TestRuby() {
	suite.OnlyRunForTags(tagsuite.Shell)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI-Testing/Ruby", "72fadc10-ed8c-4be6-810b-b3de6e017c57")

	cp := ts.Spawn("shell")
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectInput()
	cp.SendLine("ruby -v")
	cp.Expect("ActiveState")
}

func (suite *ShellIntegrationTestSuite) TestNestedShellNotification() {
	if runtime.GOOS == "windows" {
		return // cmd.exe does not have an RC file to check for nested shells in
	}
	suite.OnlyRunForTags(tagsuite.Shell)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	var ss subshell.SubShell
	var rcFile string
	env := []string{}
	switch runtime.GOOS {
	case "darwin":
		ss = &zsh.SubShell{}
		ss.SetBinary("zsh")
		rcFile = filepath.Join(ts.Dirs.HomeDir, ".zshrc")
		suite.Require().NoError(sscommon.WriteRcFile("zshrc_append.sh", rcFile, sscommon.DefaultID, nil))
		env = append(env, "SHELL=zsh") // override since CI tests are running on bash
	case "linux":
		ss = &bash.SubShell{}
		ss.SetBinary("bash")
		rcFile = filepath.Join(ts.Dirs.HomeDir, ".bashrc")
		suite.Require().NoError(sscommon.WriteRcFile("bashrc_append.sh", rcFile, sscommon.DefaultID, nil))
	default:
		suite.Fail("Unsupported OS")
	}
	suite.Require().Equal(filepath.Dir(rcFile), ts.Dirs.HomeDir, "rc file not in test suite homedir")
	suite.Require().Contains(string(fileutils.ReadFileUnsafe(rcFile)), "State Tool is operating on project")

	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("shell", "Empty"),
		e2e.OptAppendEnv(env...))
	cp.Expect("Activated")
	suite.Assert().NotContains(cp.Snapshot(), "State Tool is operating on project")
	cp.SendLine(fmt.Sprintf(`export HOME="%s"`, ts.Dirs.HomeDir)) // some shells do not forward this

	cp.SendLine(ss.Binary()) // platform-specific shell (zsh on macOS, bash on Linux, etc.)
	cp.Expect("State Tool is operating on project ActiveState-CLI/Empty")
	cp.SendLine("exit") // subshell within a subshell
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ShellIntegrationTestSuite) TestPs1() {
	if runtime.GOOS == "windows" {
		return // cmd.exe does not have a PS1 to modify
	}
	suite.OnlyRunForTags(tagsuite.Shell)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("shell", "Empty"),
	)
	cp.Expect("Activated")
	cp.Expect("[ActiveState-CLI/Empty]")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("config", "set", constants.PreservePs1ConfigKey, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("shell", "Empty")
	cp.Expect("Activated")
	suite.Assert().NotContains(cp.Snapshot(), "[ActiveState-CLI/Empty]")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ShellIntegrationTestSuite) TestProjectOrder() {
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Shell)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// First, set up a new project with a subproject.
	cp := ts.Spawn("checkout", "ActiveState-CLI/Empty", "project")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)
	projectDir := filepath.Join(ts.Dirs.Work, "project")

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/Empty", "subproject"),
		e2e.OptWD(projectDir),
	)
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)
	subprojectDir := filepath.Join(projectDir, "subproject")

	// Then set up a separate project and make it the default.
	cp = ts.Spawn("checkout", "ActiveState-CLI/Empty", "default")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)
	defaultDir := filepath.Join(ts.Dirs.Work, "default")

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("use"),
		e2e.OptWD(defaultDir),
	)
	cp.Expect("Switched to project", e2e.RuntimeSourcingTimeoutOpt)
	cp.Expect(defaultDir)
	cp.ExpectExitCode(0)

	// Now set up an empty directory.
	emptyDir := filepath.Join(ts.Dirs.Work, "empty")
	suite.Require().NoError(fileutils.Mkdir(emptyDir))

	// Now change to the project directory and assert that project is used instead of the default
	// project.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("refresh"),
		e2e.OptWD(projectDir),
	)
	cp.Expect(projectDir)
	cp.ExpectExit()

	// Run `state shell` in this project, change to the subproject directory, and assert the parent
	// project is used instead of the subproject.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("shell"),
		e2e.OptWD(projectDir),
	)
	cp.Expect("Opening shell", e2e.RuntimeSourcingTimeoutOpt)
	cp.Expect(projectDir)
	cp.SendLine("cd subproject")
	cp.SendLine("state refresh")
	cp.Expect(projectDir) // not subprojectDir
	cp.SendLine("exit")
	// cp.Expect("Deactivated") // Disabled for now due to https://activestatef.atlassian.net/browse/DX-2901
	cp.ExpectExit() // exit code varies depending on shell; just assert the shell exited

	// After exiting the shell, assert the subproject is used instead of the parent project.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("refresh"),
		e2e.OptWD(subprojectDir),
	)
	cp.Expect(subprojectDir)
	cp.ExpectExit()

	// If a project subdirectory does not contain an activestate.yaml file, assert the project that
	// owns the subdirectory will be used.
	nestedDir := filepath.Join(subprojectDir, "nested")
	suite.Require().NoError(fileutils.Mkdir(nestedDir))
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("refresh"),
		e2e.OptWD(nestedDir),
	)
	cp.Expect(subprojectDir)
	cp.ExpectExit()

	// Change to an empty directory and assert the default project is used.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("refresh"),
		e2e.OptWD(emptyDir),
	)
	cp.Expect(defaultDir)
	cp.ExpectExit()

	// If none of the above, assert an error.
	cp = ts.Spawn("use", "reset", "-n")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("refresh"),
		e2e.OptWD(emptyDir),
	)
	cp.ExpectExit()
}

func (suite *ShellIntegrationTestSuite) TestScriptAlias() {
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Shell)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Perl-5.32", ".")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	suite.NoError(fileutils.WriteFile(filepath.Join(ts.Dirs.Work, "testargs.pl"), []byte(`
printf "Argument: '%s'.\n", $ARGV[0];
`)))

	// Append a run script to activestate.yaml.
	asyFilename := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)
	contents := string(fileutils.ReadFileUnsafe(asyFilename))
	lang := "bash"
	splat := "$@"
	if runtime.GOOS == "windows" {
		lang = "powershell"
		splat = "@args"
	}
	contents = strings.Replace(contents, "events:", fmt.Sprintf(`
  - name: args
    language: %s
    value: perl testargs.pl %s

events:`, lang, splat), 1)
	suite.Require().NoError(fileutils.WriteFile(asyFilename, []byte(contents)))

	// Verify that running a script as a command with an argument containing special characters works.
	cp = ts.Spawn("shell")
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectInput()
	cp.SendLine(`args "<3"`)
	cp.Expect("Argument: '<3'", termtest.OptExpectTimeout(5*time.Second))
	cp.SendLine("exit")
	cp.Expect("Deactivated")
	cp.ExpectExit() // exit code varies depending on shell; just assert the shell exited
}

func (suite *ShellIntegrationTestSuite) TestWindowsShells() {
	if runtime.GOOS != "windows" {
		suite.T().Skip("Windows only test")
	}

	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Shell)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Empty", "6d79f2ae-f8b5-46bd-917a-d4b2558ec7b8")

	hostname, err := os.Hostname()
	suite.Require().NoError(err)
	cp := ts.SpawnCmdWithOpts(
		"cmd",
		e2e.OptArgs("/C", "state", "shell"),
		e2e.OptAppendEnv(constants.OverrideShellEnvVarName+"="),
	)
	cp.ExpectInput()
	cp.SendLine("hostname")
	cp.Expect(hostname) // cmd.exe shows the actual hostname
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// Clear configured shell.
	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	err = cfg.Set(subshell.ConfigKeyShell, "")
	suite.Require().NoError(err)

	cp = ts.SpawnCmdWithOpts(
		"powershell",
		e2e.OptArgs("-Command", "state", "shell"),
		e2e.OptAppendEnv(constants.OverrideShellEnvVarName+"="),
	)
	cp.ExpectInput()
	cp.SendLine("$host.name")
	cp.Expect("ConsoleHost") // powershell always shows ConsoleHost, go figure
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func TestShellIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ShellIntegrationTestSuite))
}
