package integration

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/suite"
)

type InstallerIntegrationTestSuite struct {
	tagsuite.Suite
	installerExe string
}

func (suite *InstallerIntegrationTestSuite) TestInstallFromLocalSource() {
	suite.OnlyRunForTags(tagsuite.Installer, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.setupTest(ts)
	suite.SetupRCFile(ts)

	target := filepath.Join(ts.Dirs.Work, "installation")

	dir, err := ioutil.TempDir("", "system*")
	suite.NoError(err)

	// Run installer with source-path flag (ie. install from this local path)
	cp := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.WithArgs(target),
		e2e.AppendEnv(constants.DisableUpdates+"=false"),
		e2e.AppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
	)

	// Assert output
	cp.Expect("Installing State Tool")
	cp.Expect("Done")
	cp.Expect("successfully installed")
	if runtime.GOOS == "darwin" && condition.OnCI() {
		cp.Expect("You are running bash on macOS")
	}
	suite.NotContains(cp.TrimmedSnapshot(), "Downloading State Tool")
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// Ensure installing overtop doesn't result in errors
	cp = ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.WithArgs(target, "--force"),
		e2e.AppendEnv(constants.DisableUpdates+"=false"),
		e2e.AppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
	)

	// Assert output
	cp.Expect("successfully installed")

	stateExec, err := installation.StateExecFromDir(target)
	suite.Contains(stateExec, target, "Ensure we're not grabbing state tool from integration test bin dir")
	suite.NoError(err)

	stateExecResolved, err := fileutils.ResolvePath(stateExec)
	suite.Require().NoError(err)

	serviceExec, err := installation.ServiceExecFromDir(target)
	suite.NoError(err)

	// Verify that launched subshell has State tool on PATH
	cp.WaitForInput()
	cp.SendLine("state --version")
	cp.Expect("Version")
	cp.WaitForInput()

	if runtime.GOOS == "windows" {
		cp.SendLine("where state")
	} else {
		cp.SendLine("which state")
	}
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	snapshot := strings.Replace(cp.TrimmedSnapshot(), "\n", "", -1)
	if !strings.Contains(snapshot, stateExec) && !strings.Contains(snapshot, stateExecResolved) {
		suite.Fail(fmt.Sprintf("Snapshot does not include '%s' or '%s', snapshot:\n %s", stateExec, stateExecResolved, snapshot))
	}

	// Assert expected files were installed (note this didn't use an update payload, so there's no bin directory)
	suite.FileExists(stateExec)
	suite.FileExists(serviceExec)

	// Run state tool so test doesn't panic trying to find the log file
	cp = ts.SpawnCmd(stateExec, "--version")
	cp.Expect("Version")

	// Assert that the config was written (ie. RC files or windows registry)
	suite.AssertConfig(ts)
}

func (suite *InstallerIntegrationTestSuite) TestInstallIncompatible() {
	if runtime.GOOS != "windows" {
		suite.T().Skip("Only Windows has incompatibility logic")
	}
	suite.OnlyRunForTags(tagsuite.Installer, tagsuite.Compatibility, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.setupTest(ts)

	target := filepath.Join(ts.Dirs.Work, "installation")

	// Run installer with source-path flag (ie. install from this local path)
	cp := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.WithArgs(target),
		e2e.AppendEnv(constants.DisableUpdates+"=false", sysinfo.VersionOverrideEnvVar+"=10.0.0"),
	)

	// Assert output
	cp.Expect("not compatible")
	cp.ExpectExitCode(1)
}

func (suite *InstallerIntegrationTestSuite) TestInstallNoErrorTips() {
	suite.OnlyRunForTags(tagsuite.Installer, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.setupTest(ts)

	target := filepath.Join(ts.Dirs.Work, "installation")

	dir, err := ioutil.TempDir("", "system*")
	suite.NoError(err)

	cp := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.WithArgs(target, "--activate", "ActiveState/DoesNotExist"),
		e2e.AppendEnv(constants.DisableUpdates+"=true"),
		e2e.AppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
	)

	cp.ExpectExitCode(1)
	suite.Assert().NotContains(cp.TrimmedSnapshot(), "Need More Help?", "error tips should not be displayed when invoking installer")
}

func (suite *InstallerIntegrationTestSuite) TestInstallErrorTips() {
	suite.OnlyRunForTags(tagsuite.Installer, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.setupTest(ts)

	target := filepath.Join(ts.Dirs.Work, "installation")

	dir, err := ioutil.TempDir("", "system*")
	suite.NoError(err)

	cp := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.WithArgs(target, "--activate", "ActiveState-CLI/Python3"),
		e2e.AppendEnv(constants.DisableUpdates+"=true"),
		e2e.AppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
	)

	cp.WaitForInput()
	cp.SendLine("state command-does-not-exist")
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.Wait()
	suite.Assert().Contains(cp.TrimmedSnapshot(), "Need More Help?", "error tips should be displayed in shell created by installer")
}

func (suite *InstallerIntegrationTestSuite) TestStateTrayRemoval() {
	suite.OnlyRunForTags(tagsuite.Installer, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.setupTest(ts)

	dir := filepath.Join(ts.Dirs.Work, "installation")

	// Install a release version that still has state-tray.
	version := "0.35.0-SHAb78e2a4"
	var cp *termtest.ConsoleProcess
	if runtime.GOOS != "windows" {
		oneLiner := fmt.Sprintf("sh <(curl -q https://platform.activestate.com/dl/cli/pdli01/install.sh) -f -n -t %s -v %s", dir, version)
		cp = ts.SpawnCmdWithOpts(
			"bash", e2e.WithArgs("-c", oneLiner),
			e2e.AppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
		)
	} else {
		b, err := download.GetDirectURL("https://platform.activestate.com/dl/cli/pdli01/install.ps1")
		suite.Require().NoError(err)

		ps1File := filepath.Join(ts.Dirs.Work, "install.ps1")
		suite.Require().NoError(fileutils.WriteFile(ps1File, b))

		cp = ts.SpawnCmdWithOpts("powershell.exe", e2e.WithArgs(ps1File, "-f", "-n", "-t", dir, "-v", version),
			e2e.AppendEnv("SHELL="),
			e2e.AppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
		)
	}
	cp.Expect("Installation Complete", 5*time.Minute)

	// Verify state-tray is there.
	svcExec, err := installation.ServiceExecFromDir(dir)
	suite.Require().NoError(err)
	trayExec := strings.Replace(svcExec, constants.StateSvcCmd, "state-tray", 1)
	suite.FileExists(trayExec)
	updateDialogExec := strings.Replace(svcExec, constants.StateSvcCmd, "state-update-dialog", 1)
	// suite.FileExists(updateDialogExec) // this is not actually installed...

	// Run the installer, which should remove state-tray and clean up after it.
	cp = ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.WithArgs("-f", "-n", "-t", dir),
		e2e.AppendEnv(constants.UpdateBranchEnvVarName+"=release"),
		e2e.AppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
	)
	cp.Expect("Installing", 10*time.Second)
	cp.Expect("Done", 30*time.Second)

	// Verify state-tray is no longer there.
	suite.NoFileExists(trayExec)
	suite.NoFileExists(updateDialogExec)

	// Verify state can still be run and has a newly updated version.
	stateExec, err := installation.StateExecFromDir(dir)
	suite.Require().NoError(err)
	cp = ts.SpawnCmdWithOpts(stateExec, e2e.WithArgs("--version"))
	suite.Assert().NotContains(cp.TrimmedSnapshot(), version)
	cp.ExpectExitCode(0)
}

func (suite *InstallerIntegrationTestSuite) TestInstallerOverwriteServiceApp() {
	suite.OnlyRunForTags(tagsuite.Installer)
	if runtime.GOOS != "darwin" {
		suite.T().Skip("Only macOS has the service app")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	appInstallDir := filepath.Join(ts.Dirs.Work, "app")
	err := fileutils.Mkdir(appInstallDir)
	suite.Require().NoError(err)

	cp := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.WithArgs(filepath.Join(ts.Dirs.Work, "installation")),
		e2e.AppendEnv(fmt.Sprintf("%s=%s", constants.AppInstallDirOverrideEnvVarName, appInstallDir)),
	)
	cp.Expect("Done")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// State Service.app should be overwritten cleanly without error.
	cp = ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.WithArgs(filepath.Join(ts.Dirs.Work, "installation2")),
		e2e.AppendEnv(fmt.Sprintf("%s=%s", constants.AppInstallDirOverrideEnvVarName, appInstallDir)),
	)
	cp.Expect("Done")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *InstallerIntegrationTestSuite) SetupRCFile(ts *e2e.Session) {
	if runtime.GOOS == "windows" {
		return
	}

	cfg, err := config.New()
	suite.Require().NoError(err)

	subshell := subshell.New(cfg)
	rcFile, err := subshell.RcFile()
	suite.Require().NoError(err)

	err = fileutils.CopyFile(rcFile, filepath.Join(ts.Dirs.HomeDir, filepath.Base(rcFile)))
	suite.Require().NoError(err)
}

func (suite *InstallerIntegrationTestSuite) AssertConfig(ts *e2e.Session) {
	if runtime.GOOS != "windows" {
		// Test bashrc
		cfg, err := config.New()
		suite.Require().NoError(err)

		subshell := subshell.New(cfg)
		rcFile, err := subshell.RcFile()
		suite.Require().NoError(err)

		if fileutils.FileExists(filepath.Join(ts.Dirs.HomeDir, filepath.Base(rcFile))) {
			rcFile = filepath.Join(ts.Dirs.HomeDir, filepath.Base(rcFile))
		}
		bashContents := fileutils.ReadFileUnsafe(rcFile)
		suite.Contains(string(bashContents), constants.RCAppendInstallStartLine, "rc file should contain our RC Append Start line")
		suite.Contains(string(bashContents), constants.RCAppendInstallStopLine, "rc file should contain our RC Append Stop line")
		suite.Contains(string(bashContents), filepath.Join(ts.Dirs.Work), "rc file should contain our target dir")
	} else {
		// Test registry
		out, err := exec.Command("reg", "query", `HKLM\SYSTEM\ControlSet001\Control\Session Manager\Environment`, "/v", "Path").Output()
		suite.Require().NoError(err)

		// we need to look for  the short and the long version of the target PATH, because Windows translates between them arbitrarily
		shortPath, err := fileutils.GetShortPathName(ts.Dirs.Work)
		suite.Require().NoError(err)
		longPath, err := fileutils.GetLongPathName(ts.Dirs.Work)
		suite.Require().NoError(err)
		if !strings.Contains(string(out), shortPath) && !strings.Contains(string(out), longPath) && !strings.Contains(string(out), ts.Dirs.Work) {
			suite.T().Errorf("registry PATH \"%s\" does not contain \"%s\", \"%s\" or \"%s\"", out, ts.Dirs.Work, shortPath, longPath)
		}
	}
}

func (s *InstallerIntegrationTestSuite) setupTest(ts *e2e.Session) {
	root := environment.GetRootPathUnsafe()
	buildDir := fileutils.Join(root, "build")
	installerExe := filepath.Join(buildDir, constants.StateInstallerCmd+osutils.ExeExt)
	if !fileutils.FileExists(installerExe) {
		s.T().Fatal("E2E tests require a state-installer binary. Run `state run build-installer`.")
	}
	s.installerExe = ts.CopyExeToDir(installerExe, filepath.Join(ts.Dirs.Base, "installer"))

	payloadDir := filepath.Dir(s.installerExe)
	ts.CopyExeToDir(ts.Exe, filepath.Join(payloadDir, installation.BinDirName))
	ts.CopyExeToDir(ts.SvcExe, filepath.Join(payloadDir, installation.BinDirName))
}

func TestInstallerIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InstallerIntegrationTestSuite))
}
