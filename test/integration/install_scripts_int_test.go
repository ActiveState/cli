package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type OpenKeyFn func(path string) (osutils.RegistryKey, error)

type CmdEnv struct {
	openKeyFn OpenKeyFn
	// whether this updates the system environment
	userScope bool
}

func NewCmdEnv(userScope bool) *CmdEnv {
	openKeyFn := osutils.OpenSystemKey
	if userScope {
		openKeyFn = osutils.OpenUserKey
	}
	return &CmdEnv{
		openKeyFn: openKeyFn,
		userScope: userScope,
	}
}

func (c *CmdEnv) set(name, newValue string) error {
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

func (c *CmdEnv) get(name string) (string, error) {
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

	cmdEnv := NewCmdEnv(!isAdmin)
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

func (suite *InstallScriptsIntegrationTestSuite) TestInstallPerl5_32() {
	if runtime.GOOS != "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T())

	isAdmin, err := osutils.IsWindowsAdmin()
	suite.Require().NoError(err, "Could not determine if running as administrator")

	cmdEnv := NewCmdEnv(!isAdmin)
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
	suite.Assert().Contains(paths, filepath.Join(ts.Dirs.Cache, "bin"), "Could not find global binary directory on PATH")
	suite.Assert().Contains(paths, ts.Dirs.Work, "Could not find installation path in PATH")
}
func TestInstallScriptsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InstallScriptsIntegrationTestSuite))
}
