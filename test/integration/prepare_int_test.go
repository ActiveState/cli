package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/platform/runtime/executor"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	rt "github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/stretchr/testify/suite"
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

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("_prepare"),
		// e2e.AppendEnv(fmt.Sprintf("ACTIVESTATE_CLI_CONFIGDIR=%s", ts.Dirs.Work)),
	)
	cp.ExpectExitCode(0)

	isAdmin, err := osutils.IsAdmin()
	suite.Require().NoError(err, "Could not determine if we are a Windows Administrator")
	// For Windows Administrator users `state _prepare` is doing nothing now (because it doesn't make sense...)
	if isAdmin {
		return
	}
	suite.AssertConfig(filepath.Join(ts.Dirs.Cache, "bin"))
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
	ts := e2e.New(suite.T(), true, "ACTIVESTATE_CLI_DISABLE_RUNTIME=false")
	err := ts.ClearCache()
	suite.Require().NoError(err)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/small-python", "--path", ts.Dirs.Work, "--default"),
	)
	cp.ExpectLongString("This project will be set as the default")
	cp.Expect("Downloading")
	cp.Expect("Installing")
	cp.Expect("Activated")

	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	suite.Require().Equal(ts.Dirs.Work, cfg.GetString(constants.GlobalDefaultPrefname))
	suite.Require().NoError(cfg.Close())

	// Remove global executors
	globalExecDir := filepath.Join(ts.Dirs.Cache, "bin")
	os.RemoveAll(globalExecDir)

	// check existens of exec dir
	targetDir := rt.ProjectDirToTargetDir(ts.Dirs.Work, ts.Dirs.Cache)
	projectExecDir := setup.ExecDir(targetDir)
	suite.DirExists(projectExecDir)

	suite.Assert().NoError(err, "should have removed executor directory, to ensure that it gets re-created")

	cp = ts.Spawn("_prepare")
	cp.ExpectExitCode(0)

	// remove complete marker to force re-creation of executors
	err = os.Remove(filepath.Join(targetDir, constants.LocalRuntimeEnvironmentDirectory, constants.RuntimeInstallationCompleteMarker))
	suite.Assert().NoError(err, "removal of complete marker should have worked")

	suite.FileExists(filepath.Join(globalExecDir, executor.NameForExe("python3"+osutils.ExeExt)))
	err = os.RemoveAll(projectExecDir)

	cp = ts.Spawn("activate")
	cp.Expect("Activated")
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
