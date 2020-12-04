package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type openKeyFn func(path string) (osutils.RegistryKey, error)

type cmdEnv struct {
	openKeyFn openKeyFn
	// whether this updates the system environment
	userScope bool
}

func newCmdEnv(userScope bool) *cmdEnv {
	openKeyFn := osutils.OpenSystemKey
	if userScope {
		openKeyFn = osutils.OpenUserKey
	}
	return &cmdEnv{
		openKeyFn: openKeyFn,
		userScope: userScope,
	}
}

func (c *cmdEnv) set(name, newValue string) error {
	key, err := c.openKeyFn(getEnvironmentPath(c.userScope))
	if err != nil {
		return err
	}
	defer key.Close()

	_, valType, err := key.GetStringValue(name)
	if err != nil {
		return err
	}
	return osutils.SetStringValue(key, name, valType, newValue)
}

func (c *cmdEnv) get(name string) (string, error) {
	key, err := c.openKeyFn(getEnvironmentPath(c.userScope))
	if err != nil {
		return "", err
	}
	defer key.Close()

	v, _, err := key.GetStringValue(name)
	return v, err
}

func getEnvironmentPath(userScope bool) string {
	if userScope {
		return "Environment"
	}
	return `SYSTEM\ControlSet001\Control\Session Manager\Environment`
}

func scriptPath(t *testing.T) string {
	name := "install.ps1"
	if runtime.GOOS != "windows" {
		name = "install.sh"
	}
	root := environment.GetRootPathUnsafe()
	subdir := "installers"

	exec := filepath.Join(root, subdir, name)
	if !fileutils.FileExists(exec) {
		t.Fatalf("Could not find install script %s", exec)
	}

	return exec
}

type InstallScriptsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallSh() {
	if runtime.GOOS == "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T())

	cp := ts.SpawnCmdWithOpts("bash", e2e.WithArgs(script, "-t", ts.Dirs.Work))
	cp.Expect("Installing to")
	cp.Expect("Continue?")
	cp.SendLine("y")
	cp.Expect("Fetching the latest version")
	cp.Expect("Allow $PATH to be appended in your")
	cp.SendLine("n")
	cp.Expect("State Tool Installed")
	cp.ExpectExitCode(0)
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallPerl5_32() {
	if runtime.GOOS != "linux" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T())

	cp := ts.SpawnCmdWithOpts(
		"bash",
		e2e.WithArgs("-c", fmt.Sprintf("cp $HOME/.bashrc %s/bashrc.bak", ts.Dirs.Work)),
	)
	cp.ExpectExitCode(0)

	defer func() {
		cp = ts.SpawnCmdWithOpts(
			"bash",
			e2e.WithArgs("-c", fmt.Sprintf("cp %s/.bashrc.bak $HOME/.bashrc", ts.Dirs.Work)),
		)
	}()

	cp.ExpectExitCode(0)
	cp = ts.SpawnCmdWithOpts(
		"bash",
		e2e.WithArgs(script, "-t", ts.Dirs.Work, "--activate-default", "ActiveState/Perl-5.32"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false", "SHELL=bash"),
	)
	cp.Expect("Installing to")
	cp.Expect("Continue?")
	cp.SendLine("y")
	cp.Expect("Fetching the latest version", 20*time.Second)
	cp.Expect("Allow $PATH to be appended in your")
	cp.SendLine("y")
	cp.Expect("State Tool installation complete")
	cp.Expect("Activating Virtual Environment")
	cp.Expect("Choose Destination")
	cp.Send("")
	cp.Expect("Cloning Repository")
	cp.Expect("Downloading missing artifacts")
	cp.Expect("Updating missing artifacts", 20*time.Second)
	cp.ExpectLongString("Successfully configured ActiveState/Perl-5.32 as the global default project")
	cp.Expect("activated state")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// we need to run an interactive bash session to ensure that the modified ~/.bashrc is being parsed
	cp = ts.SpawnCmd("bash")
	cp.SendLine("echo $PATH; exit")
	// Expect Global Binary directory on PATH
	globalBinDir := filepath.Join(ts.Dirs.Cache, "bin")
	cp.ExpectLongString(globalBinDir, 1*time.Second)
	// expect State Tool Installation directory
	cp.ExpectLongString(ts.Dirs.Work, 1*time.Second)
	cp.ExpectExitCode(0)
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallPs1() {
	if runtime.GOOS != "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T())

	isAdmin, err := osutils.IsWindowsAdmin()
	suite.Require().NoError(err, "Could not determine if running as administrator")

	cmdEnv := newCmdEnv(!isAdmin)
	oldPathEnv, err := cmdEnv.get("PATH")
	suite.Require().NoError(err, "could not get PATH")

	defer func() {
		err := cmdEnv.set("PATH", oldPathEnv)
		suite.Assert().NoError(err, "Unexpected error re-setting paths")
	}()

	cp := ts.SpawnCmdWithOpts("powershell.exe", e2e.WithArgs(script, "-t", ts.Dirs.Work))
	cp.Expect("Installing to")
	cp.Expect("Continue?")

	cp.SendLine("y")
	cp.Expect("Fetching the latest version")
	cp.Expect("State Tool successfully installed to")
	cp.ExpectExitCode(0)

	pathEnv, err := cmdEnv.get("PATH")
	suite.Require().NoError(err, "could not get PATH")
	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	suite.Assert().Contains(paths, ts.Dirs.Work, "Could not find installation path in PATH")
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallPerl5_32_Windows() {
	if runtime.GOOS != "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T())

	isAdmin, err := osutils.IsWindowsAdmin()
	suite.Require().NoError(err, "Could not determine if running as administrator")

	cmdEnv := newCmdEnv(!isAdmin)
	oldPathEnv, err := cmdEnv.get("PATH")
	suite.Require().NoError(err, "could not get PATH")

	defer func() {
		err := cmdEnv.set("PATH", oldPathEnv)
		suite.Assert().NoError(err, "Unexpected error re-setting paths")
	}()

	cp := ts.SpawnCmdWithOpts(
		"powershell.exe",
		e2e.WithArgs(script, "-t", ts.Dirs.Work, "-activate-default", "ActiveState/Perl-5.32"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"))
	cp.Expect("Installing to")
	cp.Expect("Continue?")

	cp.SendLine("y")
	cp.Expect("Fetching the latest version")
	cp.Expect("State Tool successfully installed to")
	cp.Expect("Activating project ActiveState/Perl-5.32 as default")
	cp.Expect("Cloning Repository")
	cp.Expect("Downloading missing artifacts")
	cp.Expect("Updating missing artifacts")
	cp.ExpectLongString("Successfully configured ActiveState/Perl-5.32 as the global default project")
	cp.Expect("activated state")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	pathEnv, err := cmdEnv.get("PATH")
	suite.Require().NoError(err, "could not get PATH")
	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	userPaths := paths
	if isAdmin {
		// `state prepare` writes the global bin directory to the USER path
		userCmdEnv := newCmdEnv(true)
		userPathEnv, err := userCmdEnv.get("PATH")
		suite.Require().NoError(err, "could not get PATH")
		userPaths = strings.Split(userPathEnv, string(os.PathListSeparator))
	}
	suite.Assert().Contains(userPaths, filepath.Join(ts.Dirs.Cache, "bin"), "Could not find global binary directory on PATH")
	suite.Assert().Contains(paths, ts.Dirs.Work, "Could not find installation path in PATH")
}
func TestInstallScriptsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InstallScriptsIntegrationTestSuite))
}
