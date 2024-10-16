package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type UninstallIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *UninstallIntegrationTestSuite) TestUninstall() {
	suite.OnlyRunForTags(tagsuite.Uninstall, tagsuite.Critical)
	// suite.T().Run("Partial uninstall", func(t *testing.T) { suite.testUninstall(false) })
	suite.T().Run("Full uninstall", func(t *testing.T) { suite.testUninstall(true) })
}

func (suite *UninstallIntegrationTestSuite) install(ts *e2e.Session) string {
	// Determine URL of install script.
	baseUrl := "https://state-tool.s3.amazonaws.com/update/state/"
	scriptBaseName := "install."
	if runtime.GOOS != "windows" {
		scriptBaseName += "sh"
	} else {
		scriptBaseName += "ps1"
	}
	scriptUrl := baseUrl + constants.ChannelName + "/" + scriptBaseName

	// Fetch it.
	b, err := httputil.GetDirect(scriptUrl)
	suite.Require().NoError(err)
	script := filepath.Join(ts.Dirs.Work, scriptBaseName)
	suite.Require().NoError(fileutils.WriteFile(script, b))

	// Make the directory to install to.
	appInstallDir := filepath.Join(ts.Dirs.Work, "app")
	suite.NoError(fileutils.Mkdir(appInstallDir))

	// Perform the installation.
	cmd := "bash"
	opts := []e2e.SpawnOptSetter{
		e2e.OptArgs(script, appInstallDir, "-n"),
		e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.AppInstallDirOverrideEnvVarName, appInstallDir)),
		e2e.OptAppendEnv(fmt.Sprintf("%s=FOO", constants.OverrideSessionTokenEnvVarName)),
	}
	if runtime.GOOS == "windows" {
		cmd = "powershell.exe"
		opts = append(opts, e2e.OptAppendEnv("SHELL="))
	}
	cp := ts.SpawnCmdWithOpts(cmd, opts...)
	cp.Expect("Installation Complete", e2e.RuntimeSourcingTimeoutOpt)
	cp.SendLine("exit")
	cp.ExpectExit() // exit code differs depending on shell; just assert the process exited

	return appInstallDir
}

func (suite *UninstallIntegrationTestSuite) testUninstall(all bool) {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	appInstallDir := suite.install(ts)
	binDir := filepath.Join(appInstallDir, "bin")
	stateExe := filepath.Join(binDir, filepath.Base(ts.Exe))
	svcExe := filepath.Join(binDir, filepath.Base(ts.SvcExe))

	if runtime.GOOS == "linux" {
		// When installed in a non-desktop environment (i.e. on a server), verify the user's ~/.profile was changed.
		profile := filepath.Join(ts.Dirs.HomeDir, ".profile")
		suite.Contains(string(fileutils.ReadFileUnsafe(profile)), svcExe, "autostart should be configured for Linux server environment")
	}

	cp := ts.SpawnCmdWithOpts(svcExe, e2e.OptArgs("start"))
	cp.ExpectExitCode(0)

	args := []string{"clean", "uninstall"}
	if all {
		args = append(args, "--all")
	}
	cp = ts.SpawnCmdWithOpts(
		stateExe,
		e2e.OptArgs(args...),
	)
	cp.Expect("You are about to remove")
	if !all {
		cp.Expect("--all") // verify mention of "--all" to remove everything
	}
	cp.SendLine("y")
	if runtime.GOOS == "windows" {
		cp.Expect("Deletion of State Tool has been scheduled")
	} else {
		cp.Expect("Successfully removed State Tool and related files")
	}
	cp.ExpectExitCode(0)

	if runtime.GOOS == "windows" {
		// Allow time for spawned script to remove directories
		time.Sleep(2 * time.Second)
	}

	snapshot := cp.Snapshot()
	fmt.Println(snapshot)

	if all {
		suite.NoDirExists(ts.Dirs.Cache, "Cache dir should not exist after full uninstall")
		suite.NoDirExists(ts.Dirs.Config, "Config dir should not exist after full uninstall")
	} else {
		suite.DirExists(ts.Dirs.Cache, "Cache dir should still exist after partial uninstall")
		suite.DirExists(ts.Dirs.Config, "Config dir should still exist after partial uninstall")
	}

	if fileutils.FileExists(stateExe) {
		suite.Fail("State tool executable should not exist after uninstall")
	}

	if fileutils.FileExists(svcExe) {
		suite.Fail("State service executable should not exist after uninstall")
	}

	if runtime.GOOS == "linux" {
		// When installed in a non-desktop environment (i.e. on a server), verify the user's ~/.profile was reverted.
		homeDir, err := user.HomeDir()
		suite.Require().NoError(err)
		profile := filepath.Join(homeDir, ".profile")
		suite.NotContains(string(fileutils.ReadFileUnsafe(profile)), svcExe, "autostart should not be configured for Linux server environment anymore")
	}

	if runtime.GOOS == "darwin" {
		if fileutils.DirExists(filepath.Join(binDir, "system")) {
			suite.Fail("system directory should not exist after uninstall")
		}
	}

	if runtime.GOOS == "windows" {
		shortcutDir := filepath.Join(ts.Dirs.HomeDir, "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "ActiveState")
		suite.NoDirExists(shortcutDir, "shortcut dir should not exist after uninstall")
	}

	if fileutils.DirExists(binDir) {
		suite.Fail("bin directory should not exist after uninstall")
	}
}

func TestUninstallIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UninstallIntegrationTestSuite))
}
