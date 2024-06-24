package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"

	svcApp "github.com/ActiveState/cli/cmd/state-svc/app"
	svcAutostart "github.com/ActiveState/cli/cmd/state-svc/autostart"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	rt "github.com/ActiveState/cli/pkg/platform/runtime/target"
)

type PrepareIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PrepareIntegrationTestSuite) TestPrepare() {
	// Disable test for v0.36: https://activestatef.atlassian.net/browse/DX-1501.
	// This test should be re-enabled by https://activestatef.atlassian.net/browse/DX-1435.
	suite.T().SkipNow()

	suite.OnlyRunForTags(tagsuite.Prepare)
	if !e2e.RunningOnCI() {
		suite.T().Skipf("Skipping TestPrepare when not running on CI or on MacOS, as it modifies PATH")
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	autostartDir := filepath.Join(ts.Dirs.Work, "autostart")
	err := fileutils.Mkdir(autostartDir)
	suite.Require().NoError(err)

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("_prepare"),
		e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.AutostartPathOverrideEnvVarName, autostartDir)),
		// e2e.OptAppendEnv(fmt.Sprintf("%s=%s", constants.ConfigEnvVarName, ts.Dirs.Work)),
	)
	cp.ExpectExitCode(0)

	isAdmin, err := osutils.IsAdmin()
	suite.Require().NoError(err, "Could not determine if we are a Windows Administrator")
	// For Windows Administrator users `state _prepare` is doing nothing now (because it doesn't make sense...)
	if isAdmin {
		return
	}
	suite.AssertConfig(filepath.Join(ts.Dirs.Cache, "bin"))

	// Verify autostart was enabled.
	app, err := svcApp.New()
	suite.Require().NoError(err)
	enabled, err := autostart.IsEnabled(app.Path(), svcAutostart.Options)
	suite.Require().NoError(err)
	suite.Assert().True(enabled, "autostart is not enabled")

	// When installed in a non-desktop environment (i.e. on a server), verify the user's ~/.profile was amended.
	if runtime.GOOS == "linux" {
		homeDir, err := user.HomeDir()
		suite.Require().NoError(err)
		profile := filepath.Join(homeDir, ".profile")
		profileContents := string(fileutils.ReadFileUnsafe(profile))
		suite.Contains(profileContents, app.Path(), "autostart should be configured for Linux server environment")
	}

	// Verify autostart can be disabled.
	err = autostart.Disable(app.Path(), svcAutostart.Options)
	suite.Require().NoError(err)
	enabled, err = autostart.IsEnabled(app.Path(), svcAutostart.Options)
	suite.Require().NoError(err)
	suite.Assert().False(enabled, "autostart is still enabled")

	// When installed in a non-desktop environment (i.e. on a server), verify the user's ~/.profile was reverted.
	if runtime.GOOS == "linux" {
		homeDir, err := user.HomeDir()
		suite.Require().NoError(err)
		profile := filepath.Join(homeDir, ".profile")
		profileContents := fileutils.ReadFileUnsafe(profile)
		suite.NotContains(profileContents, app.Exec, "autostart should not be configured for Linux server environment anymore")
	}
}

func (suite *PrepareIntegrationTestSuite) AssertConfig(target string) {
	if runtime.GOOS != "windows" {
		// Test config file
		cfg, err := config.New()
		suite.Require().NoError(err)

		subshell := subshell.New(cfg)
		rcFile, err := subshell.RcFile()
		suite.Require().NoError(err)

		bashContents := fileutils.ReadFileUnsafe(rcFile)
		suite.Contains(string(bashContents), constants.RCAppendDefaultStartLine, "config file should contain our RC Append Start line")
		suite.Contains(string(bashContents), constants.RCAppendDefaultStopLine, "config file should contain our RC Append Stop line")
		suite.Contains(string(bashContents), target, "config file should contain our target dir")
	} else {
		// Test registry
		out, err := exec.Command("reg", "query", `HKCU\Environment`, "/v", "Path").Output()
		suite.Require().NoError(err)
		suite.Contains(string(out), target, "Windows system PATH should contain our target dir")
	}
}

func (suite *PrepareIntegrationTestSuite) TestResetExecutors() {
	suite.OnlyRunForTags(tagsuite.Prepare)
	ts := e2e.New(suite.T(), true, constants.DisableRuntime+"=false")
	err := ts.ClearCache()
	suite.Require().NoError(err)
	defer ts.Close()

	cp := ts.Spawn("activate", "ActiveState-CLI/small-python", "--path", ts.Dirs.Work, "--default")
	cp.Expect("This project will always be available for use")
	cp.Expect("Downloading")
	cp.Expect("Installing")
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)

	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	suite.Require().Equal(ts.Dirs.Work, cfg.GetString(constants.GlobalDefaultPrefname))
	suite.Require().NoError(cfg.Close())

	// Remove global executors
	globalExecDir := filepath.Join(ts.Dirs.Cache, "bin")
	err = os.RemoveAll(globalExecDir)
	suite.Assert().NoError(err, "should have removed executor directory, to ensure that it gets re-created")

	// check existens of exec dir
	targetDir := rt.ProjectDirToTargetDir(ts.Dirs.Work, ts.Dirs.Cache)
	projectExecDir := setup.ExecDir(targetDir)
	suite.DirExists(projectExecDir)

	// remove complete marker to force re-creation of executors
	err = os.Remove(filepath.Join(targetDir, constants.LocalRuntimeEnvironmentDirectory, constants.RuntimeInstallationCompleteMarker))
	suite.Assert().NoError(err, "removal of complete marker should have worked")

	cp = ts.Spawn("_prepare")
	cp.ExpectExitCode(0)

	suite.FileExists(filepath.Join(globalExecDir, "python3"+osutils.ExeExtension))
	err = os.RemoveAll(projectExecDir)
	suite.Assert().NoError(err, "should have removed executor directory, to ensure that it gets re-created")

	cp = ts.Spawn("activate")
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.SendLine("which python3")
	cp.SendLine("python3 --version")
	cp.Expect("ActiveState")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// executor dir should be re-created
	suite.DirExists(projectExecDir)
}

func TestPrepareIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PrepareIntegrationTestSuite))
}
