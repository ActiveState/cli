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
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/stretchr/testify/suite"
)

type InstallerIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *InstallerIntegrationTestSuite) TestInstallFromLocalSource() {
	suite.OnlyRunForTags(tagsuite.Installer, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	target := filepath.Join(ts.Dirs.Work, "installation")

	// Run installer with source-path flag (ie. install from this local path)
	cp := ts.SpawnCmdWithOpts(
		ts.InstallerExe,
		e2e.WithArgs(target, "--source-path", ts.Dirs.Base),
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

	cp = ts.SpawnCmd("state", "--version")
	cp.Expect("Version")

	if runtime.GOOS == "windows" {
		cp.SendLine("where state")
	} else {
		cp.SendLine("which state")
	}

	cp.WaitForInput()
	fmt.Println("Untrimmed snapshot:", cp.Snapshot())
	snapshot := strings.Replace(cp.TrimmedSnapshot(), "\n", "", -1)
	if !strings.Contains(snapshot, stateExec) && !strings.Contains(snapshot, stateExecResolved) {
		suite.Fail(fmt.Sprintf("Snapshot does not include '%s' or '%s', snapshot:\n %s", stateExec, stateExecResolved, snapshot))
	}
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

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

	target := filepath.Join(ts.Dirs.Work, "installation")

	// Run installer with source-path flag (ie. install from this local path)
	cp := ts.SpawnCmdWithOpts(
		ts.InstallerExe,
		e2e.WithArgs(target, "--source-path", ts.Dirs.Base),
		e2e.AppendEnv(constants.DisableUpdates+"=false", sysinfo.VersionOverrideEnvVar+"=10.0.0"),
	)

	// Assert output
	cp.Expect("not compatible")
	cp.ExpectExitCode(1)
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

func TestInstallerIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InstallerIntegrationTestSuite))
}
