package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	svcAutostart "github.com/ActiveState/cli/cmd/state-svc/autostart"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/autostart"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	rt "github.com/ActiveState/cli/pkg/runtime"
	"github.com/ActiveState/cli/pkg/runtime_helpers"
)

type PrepareIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PrepareIntegrationTestSuite) TestPrepare() {
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
	)
	cp.ExpectExitCode(0)

	isAdmin, err := osutils.IsAdmin()
	suite.Require().NoError(err, "Could not determine if we are a Windows Administrator")
	// For Windows Administrator users `state _prepare` is doing nothing now (because it doesn't make sense...)
	if isAdmin {
		return
	}
	suite.AssertConfig(storage.CachePath())

	// Verify autostart was enabled.
	enabled, err := autostart.IsEnabled(constants.StateSvcCmd, svcAutostart.Options)
	suite.Require().NoError(err)
	suite.Assert().True(enabled, "autostart is not enabled")

	// When installed in a non-desktop environment (i.e. on a server), verify the user's ~/.profile was amended.
	if runtime.GOOS == "linux" {
		homeDir, err := user.HomeDir()
		suite.Require().NoError(err)
		profile := filepath.Join(homeDir, ".profile")
		profileContents := string(fileutils.ReadFileUnsafe(profile))
		suite.Contains(profileContents, constants.StateSvcCmd, "autostart should be configured for Linux server environment")
	}

	// Verify autostart can be disabled.
	err = autostart.Disable(constants.StateSvcCmd, svcAutostart.Options)
	suite.Require().NoError(err)
	enabled, err = autostart.IsEnabled(constants.StateSvcCmd, svcAutostart.Options)
	suite.Require().NoError(err)
	suite.Assert().False(enabled, "autostart is still enabled")

	// When installed in a non-desktop environment (i.e. on a server), verify the user's ~/.profile was reverted.
	if runtime.GOOS == "linux" {
		homeDir, err := user.HomeDir()
		suite.Require().NoError(err)
		profile := filepath.Join(homeDir, ".profile")
		profileContents := fileutils.ReadFileUnsafe(profile)
		suite.NotContains(profileContents, constants.StateSvcCmd, "autostart should not be configured for Linux server environment anymore")
	}

	// Verify the Windows shortcuts were installed.
	if runtime.GOOS == "windows" {
		shortcutDir := filepath.Join(ts.Dirs.HomeDir, "AppData", "Roaming", "Microsoft", "Windows", "Start Menu", "Programs", "ActiveState")
		suite.DirExists(shortcutDir, "shortcut dir should exist after prepare")
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
	ts := e2e.New(suite.T(), false)
	err := ts.ClearCache()
	suite.Require().NoError(err)
	defer ts.Close()

	cp := ts.Spawn("activate", "ActiveState-CLI/small-python", "--path", ts.Dirs.Work, "--default")
	cp.Expect("This project will always be available for use")
	cp.Expect("Downloading")
	cp.Expect("Installing", e2e.RuntimeSourcingTimeoutOpt)
	cp.Expect("Activated")

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
	targetDir := filepath.Join(ts.Dirs.Cache, runtime_helpers.DirNameFromProjectDir(ts.Dirs.Work))
	projectExecDir := rt.ExecutorsPath(targetDir)
	suite.DirExists(projectExecDir)

	// Invalidate hash
	hashPath := filepath.Join(targetDir, ".activestate", "hash.txt")
	suite.Require().NoError(fileutils.WriteFile(hashPath, []byte("bogus")))

	cp = ts.Spawn("_prepare")
	cp.ExpectExitCode(0)

	suite.Require().FileExists(filepath.Join(globalExecDir, "python3"+osutils.ExeExtension), ts.DebugMessage(""))
	suite.Require().NoError(os.RemoveAll(projectExecDir), "should have removed executor directory, to ensure that it gets re-created")

	cp = ts.Spawn("activate")
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.SendLine("which python3")
	cp.SendLine("python3")
	cp.Expect("ActiveState")
	cp.SendLine("exit()") // exit from Python interpreter
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// executor dir should be re-created
	suite.DirExists(projectExecDir)
}

func TestPrepareIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PrepareIntegrationTestSuite))
}
