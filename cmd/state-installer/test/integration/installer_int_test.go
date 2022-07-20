package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/sysinfo"
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

	target := filepath.Join(ts.Dirs.Work, "installation")

	// Run installer with source-path flag (ie. install from this local path)
	cp := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.WithArgs(target),
		e2e.AppendEnv(constants.DisableUpdates+"=false"),
	)

	// Assert output
	cp.Expect("Installing State Tool")
	cp.Expect("Done")
	cp.Expect("successfully installed")
	suite.NotContains(cp.TrimmedSnapshot(), "Downloading State Tool")

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

	cp := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.WithArgs(target, "--activate", "ActiveState/DoesNotExist"),
		e2e.AppendEnv(constants.DisableUpdates+"=true"),
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

	cp := ts.SpawnCmdWithOpts(
		suite.installerExe,
		e2e.WithArgs(target, "--activate", "ActiveState-CLI/Python3"),
		e2e.AppendEnv(constants.DisableUpdates+"=true"),
	)

	cp.WaitForInput()
	cp.SendLine("state command-does-not-exist")
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.Wait()
	suite.Assert().Contains(cp.TrimmedSnapshot(), "Need More Help?", "error tips should be displayed in shell created by installer")
}

func (suite *InstallerIntegrationTestSuite) AssertConfig(ts *e2e.Session) {
	if runtime.GOOS != "windows" {
		// Test bashrc
		homeDir, err := os.UserHomeDir()
		suite.Require().NoError(err)

		fname := ".bashrc"
		if strings.Contains(os.Getenv("SHELL"), "zsh") {
			fname = ".zshrc"
		}

		bashContents := fileutils.ReadFileUnsafe(filepath.Join(homeDir, fname))
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
	ts.CopyExeToDir(ts.TrayExe, filepath.Join(payloadDir, installation.BinDirName))
}

func TestInstallerIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InstallerIntegrationTestSuite))
}
