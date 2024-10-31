package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/sysinfo"
)

type InstallerIntegrationTestSuite struct {
	tagsuite.Suite
	installerExe string
}

func (suite *InstallerIntegrationTestSuite) TestInstallFromLocalSource() {
	suite.OnlyRunForTags(tagsuite.Installer, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.SetupRCFile()
	suite.T().Setenv(constants.HomeEnvVarName, ts.Dirs.HomeDir)

	dir, err := os.MkdirTemp("", "system*")
	suite.NoError(err)

	// Run installer with source-path flag (ie. install from this local path)
	cp := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.OptArgs(installationDir(ts), "-n"),
		e2e.OptAppendEnv(constants.DisableUpdates+"=false"),
		e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
	)

	// Assert output
	cp.Expect("Installing State Tool")
	cp.Expect("Done")
	cp.Expect("successfully installed")
	suite.NotContains(cp.Output(), "Downloading State Tool")
	cp.ExpectInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// Ensure installing overtop doesn't result in errors
	cp = ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.OptArgs(installationDir(ts), "-n"),
		e2e.OptAppendEnv(constants.DisableUpdates+"=false"),
		e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
	)
	cp.Expect("successfully installed")
	cp.ExpectInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// Again ensure installing overtop doesn't result in errors, but mock an older state tool format where
	// the marker has no contents
	suite.Require().NoError(fileutils.WriteFile(filepath.Join(installationDir(ts), installation.InstallDirMarker), []byte{}))
	cp = ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.OptArgs(installationDir(ts), "-n"),
		e2e.OptAppendEnv(constants.DisableUpdates+"=false"),
		e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
	)
	cp.Expect("successfully installed")

	installDir := installationDir(ts)

	stateExec, err := installation.StateExecFromDir(installDir)
	suite.Contains(stateExec, installDir, "Ensure we're not grabbing state tool from integration test bin dir")
	suite.NoError(err)

	stateExecResolved, err := fileutils.ResolvePath(stateExec)
	suite.Require().NoError(err)

	serviceExec, err := installation.ServiceExecFromDir(installDir)
	suite.NoError(err)

	// Verify that launched subshell has State tool on PATH
	cp.ExpectInput()
	cp.SendLine("state --version")
	cp.Expect("Version")
	cp.ExpectInput()

	if runtime.GOOS == "windows" {
		cp.SendLine("where state")
	} else {
		cp.SendLine("which state")
	}
	cp.ExpectInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	snapshot := strings.Replace(cp.Output(), "\n", "", -1)
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

	// Run installer with source-path flag (ie. install from this local path)
	cp := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.OptArgs(installationDir(ts), "-n"),
		e2e.OptAppendEnv(constants.DisableUpdates+"=false", sysinfo.VersionOverrideEnvVar+"=10.0.0"),
	)

	// Assert output
	cp.Expect("not compatible")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *InstallerIntegrationTestSuite) TestInstallNoErrorTips() {
	suite.OnlyRunForTags(tagsuite.Installer, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	dir, err := os.MkdirTemp("", "system*")
	suite.NoError(err)

	cp := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.OptArgs(installationDir(ts), "--activate", "ActiveState/DoesNotExist", "-n"),
		e2e.OptAppendEnv(constants.DisableUpdates+"=true"),
		e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
	)

	cp.ExpectExitCode(1)
	suite.Assert().NotContains(cp.Output(), "Need More Help?", "error tips should not be displayed when invoking installer")
	ts.IgnoreLogErrors()
}

func (suite *InstallerIntegrationTestSuite) TestInstallErrorTips() {
	suite.OnlyRunForTags(tagsuite.Installer, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	dir, err := os.MkdirTemp("", "system*")
	suite.NoError(err)

	cp := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.OptArgs(installationDir(ts), "--activate", "ActiveState-CLI/Python3", "-n"),
		e2e.OptAppendEnv(constants.DisableUpdates+"=true"),
		e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
	)

	cp.ExpectInput(e2e.RuntimeSourcingTimeoutOpt)
	cp.SendLine("state command-does-not-exist")
	cp.ExpectInput()
	cp.SendLine("exit")
	cp.ExpectExit()
	suite.Assert().Contains(cp.Output(), "Need More Help?",
		"error tips should be displayed in shell created by installer")
	ts.IgnoreLogErrors()
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
		e2e.OptArgs(installationDir(ts), "-n"),
		e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.AppInstallDirOverrideEnvVarName, appInstallDir)),
	)
	cp.Expect("Done")
	cp.SendLine("exit")
	cp.ExpectExit() // the return code can vary depending on shell (e.g. zsh vs. bash); just assert the installer shell exited

	// State Service.app should be overwritten cleanly without error.
	cp = ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.OptArgs(installationDir(ts)+"2", "-n"),
		e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.AppInstallDirOverrideEnvVarName, appInstallDir)),
	)
	cp.Expect("Done")
	cp.SendLine("exit")
	cp.ExpectExit() // the return code can vary depending on shell (e.g. zsh vs. bash); just assert the installer shell exited
}

func (suite *InstallerIntegrationTestSuite) TestInstallWhileInUse() {
	suite.OnlyRunForTags(tagsuite.Installer)
	if runtime.GOOS != "windows" {
		suite.T().Skip("Only windows can have issues with copying over files in use")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	dir, err := os.MkdirTemp("", "system*")
	suite.NoError(err)

	cp := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.OptArgs(installationDir(ts), "-n"),
		e2e.OptAppendEnv(constants.DisableUpdates+"=true"),
		e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
	)
	cp.Expect("successfully installed", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectInput()
	cp.SendLine("state checkout ActiveState/Perl-5.32")
	cp.Expect("Checked out", e2e.RuntimeSourcingTimeoutOpt)
	cp.SendLine("state shell Perl-5.32")
	cp.Expect("Activated") // state.exe remains active

	// On Windows we cannot delete files/executables in use. Instead, the installer copies new
	// executables into the target directory with the ".new" suffix and renames them to the target
	// executables. Verify that this works without error.
	cp2 := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.OptArgs(installationDir(ts), "-f", "-n"),
		e2e.OptAppendEnv(constants.DisableUpdates+"=true"),
		e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir)),
	)
	cp2.Expect("successfully installed", e2e.RuntimeSourcingTimeoutOpt)
	cp2.ExpectInput()
	cp2.SendLine("exit")
	cp2.ExpectExit() // the return code can vary depending on shell (e.g. zsh vs. bash); just assert the installer shell exited

	oldStateExeFound := false
	files, err := fileutils.ListDirSimple(filepath.Join(installationDir(ts), "bin"), false)
	suite.Require().NoError(err)

	for _, file := range files {
		if strings.Contains(file, "state.exe") && strings.HasSuffix(file, ".old") {
			oldStateExeFound = true
			break
		}
	}
	suite.Assert().True(oldStateExeFound, "the state.exe currently in use was not copied to a '.old' file")

	cp.SendLine("exit") // state shell
	cp.SendLine("exit") // installer shell
	cp.ExpectExit()     // the return code can vary depending on shell (e.g. zsh vs. bash); just assert the installer shell exited
}

func (suite *InstallerIntegrationTestSuite) AssertConfig(ts *e2e.Session) {
	if runtime.GOOS != "windows" {
		// Test bashrc
		cfg, err := config.New()
		suite.Require().NoError(err)

		subshell := subshell.New(cfg)
		rcFile, err := subshell.RcFile()
		suite.Require().NoError(err)

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

func installationDir(ts *e2e.Session) string {
	return filepath.Join(ts.Dirs.Work, "installation")
}

func (suite *InstallerIntegrationTestSuite) SetupSuite() {
	rootPath := environment.GetRootPathUnsafe()
	localPayload := filepath.Join(rootPath, "build", "payload", constants.LegacyToplevelInstallArchiveDir)
	suite.Require().DirExists(localPayload, "locally generated payload exists")

	installerExe := filepath.Join(localPayload, constants.StateInstallerCmd+osutils.ExeExtension)
	suite.Require().FileExists(installerExe, "locally generated installer exists")

	suite.installerExe = installerExe
}

func TestInstallerIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InstallerIntegrationTestSuite))
}
