package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/require"
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

// scriptPath returns the path to an installation script copied to targetDir, if useTestUrl is true, the install script is modified to download from the local test server instead
func scriptPath(t *testing.T, targetDir string, legacy, useTestUrl bool) string {
	name := "install.ps1"
	if runtime.GOOS != "windows" {
		name = "install.sh"
	}
	if legacy {
		name = "legacy-" + name
	}
	root := environment.GetRootPathUnsafe()
	subdir := "installers"

	exec := filepath.Join(root, subdir, name)
	if !fileutils.FileExists(exec) {
		t.Fatalf("Could not find install script %s", exec)
	}

	b, err := fileutils.ReadFile(exec)
	require.NoError(t, err)

	if useTestUrl {
		b = bytes.Replace(b, []byte(fmt.Sprintf("%sstate", constants.APIUpdateURL)), []byte("http://localhost:"+testPort), -1)
		require.Contains(t, string(b), "http://localhost:"+testPort)
	}

	scriptPath := filepath.Join(targetDir, filepath.Base(exec))
	err = ioutil.WriteFile(scriptPath, b, 0775)
	require.NoError(t, err)

	return scriptPath
}

type InstallScriptsIntegrationTestSuite struct {
	tagsuite.Suite
	cfg    projectfile.ConfigGetter
	server *http.Server
}

func expectLegacyStateToolInstallation(cp *termtest.ConsoleProcess, addToPathAnswer string) {
	cp.Expect("Installing to")
	cp.Expect("Continue?")
	cp.SendLine("y")
	cp.Expect("Fetching the latest version")
	cp.Expect("Allow $PATH to be appended in your")
	cp.SendLine(addToPathAnswer)
	cp.Expect("State Tool installation complete")
}

func expectStateToolInstallation(cp *termtest.ConsoleProcess) {
	cp.Expect("Installing to")
	cp.Expect("Continue?")
	cp.SendLine("y")
	cp.Expect("Fetching the latest version")
	cp.Expect("State Tool installation complete")
}

func expectVersionedStateToolInstallation(cp *termtest.ConsoleProcess, version string) {
	cp.Expect("Installing to")
	cp.Expect("Continue?")
	cp.SendLine("y")
	cp.Expect(fmt.Sprintf("Fetching version: %s", version))
	cp.Expect("State Tool installation complete")
}

func expectStateToolInstallationWindows(cp *termtest.ConsoleProcess) {
	cp.Expect("Installing to")
	cp.Expect("Continue?")
	cp.SendLine("y")
	cp.Expect("Fetching the latest version")
	cp.ExpectLongString("Please start a new shell in order to start using the State Tool")
	cp.Expect("State Tool successfully installed to")
}

func expectVersionedStateToolInstallationWindows(cp *termtest.ConsoleProcess, version string) {
	cp.Expect("Installing to")
	cp.Expect("Continue?")
	cp.SendLine("y")
	cp.Expect(fmt.Sprintf("Fetching version: %s", version))
	cp.ExpectLongString("Please start a new shell in order to start using the State Tool")
	cp.Expect("State Tool successfully installed to")
}

func expectLegacyStateToolInstallationWindows(cp *termtest.ConsoleProcess) {
	cp.Expect("Installing to")
	cp.Expect("Continue?")
	cp.SendLine("y")
	cp.Expect("Fetching the latest version")
	cp.Expect("State Tool successfully installed to")
}

func expectDefaultActivation(cp *termtest.ConsoleProcess) {
	cp.Expect("Activating Virtual Environment")
	cp.Expect("Choose Destination")
	cp.Send("")
	cp.Expect("Cloning Repository")
	cp.Expect("Installing")
	cp.ExpectLongString("Successfully configured ActiveState/Perl-5.32 as the global default project")
	cp.Expect("Running Activation Events")
	cp.SendLine("exit")
}

func (suite *InstallScriptsIntegrationTestSuite) TestLegacyInstallSh() {
	if runtime.GOOS == "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T(), ts.Dirs.Work, true, false)

	cp := ts.SpawnCmdWithOpts("bash", e2e.WithArgs(script, "-t", ts.Dirs.Work))
	expectLegacyStateToolInstallation(cp, "n")
	cp.Expect("State Tool Installed")
	cp.ExpectExitCode(0)
}

func (suite *InstallScriptsIntegrationTestSuite) TestLegacyInstallShInstallMultiFileUpdate() {
	if runtime.GOOS == "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T(), ts.Dirs.Work, true, true)

	cp := ts.SpawnCmdWithOpts(
		"bash",
		e2e.WithArgs(script, "-t", ts.Dirs.Work, "-b", constants.BranchName),
		e2e.AppendEnv(fmt.Sprintf("_TEST_UPDATE_URL=http://localhost:%s/", testPort)))

	expectLegacyStateToolInstallation(cp, "n")
	cp.Expect("State Tool Installed")
	cp.ExpectExitCode(0)

	suite.FileExists(filepath.Join(ts.Dirs.Work, "state-svc"))
	suite.FileExists(filepath.Join(ts.Dirs.Work, "state-tray"))
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallSh() {
	if runtime.GOOS == "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)

	tests := []struct {
		Name        string
		TestInstall bool
		Channel     string
	}{
		{"install-local-test-update", true, constants.BranchName},
		// Todo https://www.pivotaltracker.com/story/show/177863116
		// Replace the target branch for this test to release, as soon as we have a working deployment there.
		{"install-release", false, "beta"},
	}

	for _, tt := range tests {
		if !tt.TestInstall {
			// Todo https://www.pivotaltracker.com/story/show/177858645
			suite.T().Skipf("Non-local State Tool installations will only work once we have a new State Tool installer deployed.")
		}
		suite.Run(tt.Name, func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			script := scriptPath(suite.T(), ts.Dirs.Work, false, tt.TestInstall)

			cp := ts.SpawnCmdWithOpts("bash", e2e.WithArgs(script, "-t", ts.Dirs.Work, "-b", tt.Channel))
			expectStateToolInstallation(cp)
			cp.Expect("State Tool Installed")
			cp.ExpectExitCode(0)
		})
	}
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallShVersion() {
	if runtime.GOOS == "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.InstallScripts)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	expected := "0.28.0-SHA249ab6f"
	suite.installVersion(ts, ts.Dirs.Work, expected)
	suite.compareVersionedInstall(ts, filepath.Join(ts.Dirs.Work, "state"), expected)
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallPerl5_32Default() {
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)
	suite.runInstallTest("-c", "state activate ActiveState/Perl-5.32 --default")
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallPerl5_32ActivateDefault() {
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)
	suite.runInstallTest("--activate-default", "ActiveState/Perl-5.32")
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallPs1() {
	if runtime.GOOS != "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)

	tests := []struct {
		Name        string
		TestInstall bool
		Channel     string
	}{
		{"install-local-test-update", true, constants.BranchName},
		// Todo https://www.pivotaltracker.com/story/show/177863116
		// Replace the target branch for this test to release, as soon as we have a working deployment there.
		{"install-release", false, "beta"},
	}

	for _, tt := range tests {
		if !tt.TestInstall {
			// Todo https://www.pivotaltracker.com/story/show/177858645
			suite.T().Skipf("Non-local State Tool installations will only work once we have a new State Tool installer deployed.")
		}
		suite.Run(tt.Name, func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			script := scriptPath(suite.T(), ts.Dirs.Work, false, tt.TestInstall)

			isAdmin, err := osutils.IsWindowsAdmin()
			suite.Require().NoError(err, "Could not determine if running as administrator")

			cmdEnv := newCmdEnv(!isAdmin)
			oldPathEnv, err := cmdEnv.get("PATH")
			suite.Require().NoError(err, "could not get PATH")

			defer func() {
				err := cmdEnv.set("PATH", oldPathEnv)
				suite.Assert().NoError(err, "Unexpected error re-setting paths")
			}()

			cp := ts.SpawnCmdWithOpts("powershell.exe", e2e.WithArgs(script, "-t", ts.Dirs.Work, "-b", tt.Channel), e2e.AppendEnv("SHELL="))
			expectStateToolInstallationWindows(cp)
			cp.ExpectExitCode(0)

			pathEnv, err := cmdEnv.get("PATH")
			suite.Require().NoError(err, "could not get PATH")
			paths := strings.Split(pathEnv, string(os.PathListSeparator))
			suite.Assert().Contains(paths, ts.Dirs.Work, "Could not find installation path, output: %s", cp.TrimmedSnapshot())
		})
	}
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallPs1Version() {
	if runtime.GOOS != "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.InstallScripts)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	isAdmin, err := osutils.IsWindowsAdmin()
	suite.Require().NoError(err, "Could not determine if running as administrator")

	cmdEnv := newCmdEnv(!isAdmin)
	oldPathEnv, err := cmdEnv.get("PATH")
	suite.Require().NoError(err, "could not get PATH")

	defer func() {
		err := cmdEnv.set("PATH", oldPathEnv)
		suite.Assert().NoError(err, "Unexpected error re-setting paths")
	}()

	expected := "0.28.0-SHA249ab6f"
	cp := suite.installVersion(ts, ts.Dirs.Work, expected)

	pathEnv, err := cmdEnv.get("PATH")
	suite.Require().NoError(err, "could not get PATH")
	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	suite.Assert().Contains(paths, ts.Dirs.Work, "Could not find installation path, output: %s", cp.TrimmedSnapshot())

	suite.compareVersionedInstall(ts, filepath.Join(ts.Dirs.Work, "state.exe"), expected)
}

func (suite *InstallScriptsIntegrationTestSuite) installVersion(ts *e2e.Session, target, version string) *termtest.ConsoleProcess {
	script := scriptPath(suite.T(), ts.Dirs.Work, false, false)

	shell := "bash"
	expectVersionInstall := expectVersionedStateToolInstallation
	if runtime.GOOS == "windows" {
		shell = "powershell.exe"
		expectVersionInstall = expectVersionedStateToolInstallationWindows
	}

	expected := "0.28.0-SHA249ab6f"
	cp := ts.SpawnCmdWithOpts(shell, e2e.WithArgs(script, "-t", ts.Dirs.Work, "-b", "master", "-v", expected))
	expectVersionInstall(cp, expected)
	cp.ExpectExitCode(0)

	return cp
}

func (suite *InstallScriptsIntegrationTestSuite) compareVersionedInstall(ts *e2e.Session, installPath, expected string) {
	type versionData struct {
		Version string `json:"version"`
	}

	cp := ts.SpawnCmd(installPath, "--version", "--output=json")
	cp.ExpectExitCode(0)
	actual := versionData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &actual)

	suite.Equal(expected, actual.Version)
}

func (suite *InstallScriptsIntegrationTestSuite) TestLegacyInstallPs1() {
	if runtime.GOOS != "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T(), ts.Dirs.Work, true, false)

	isAdmin, err := osutils.IsWindowsAdmin()
	suite.Require().NoError(err, "Could not determine if running as administrator")

	cmdEnv := newCmdEnv(!isAdmin)
	oldPathEnv, err := cmdEnv.get("PATH")
	suite.Require().NoError(err, "could not get PATH")

	defer func() {
		err := cmdEnv.set("PATH", oldPathEnv)
		suite.Assert().NoError(err, "Unexpected error re-setting paths")
	}()

	cp := ts.SpawnCmdWithOpts("powershell.exe", e2e.WithArgs(script, "-t", ts.Dirs.Work), e2e.AppendEnv("SHELL="))
	expectLegacyStateToolInstallationWindows(cp)
	cp.ExpectExitCode(0)

	pathEnv, err := cmdEnv.get("PATH")
	suite.Require().NoError(err, "could not get PATH")
	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	suite.Assert().Contains(paths, ts.Dirs.Work, "Could not find installation path, output: %s", cp.TrimmedSnapshot())
}

func (suite *InstallScriptsIntegrationTestSuite) TestLegacyInstallPs1MultiFileUpdate() {
	if runtime.GOOS != "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T(), ts.Dirs.Work, true, true)

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
		e2e.WithArgs(script, "-t", ts.Dirs.Work, "-b", constants.BranchName),
		e2e.AppendEnv("SHELL=", fmt.Sprintf("_TEST_UPDATE_URL=http://localhost:%s/", testPort)))

	expectLegacyStateToolInstallationWindows(cp)
	cp.ExpectExitCode(0)

	pathEnv, err := cmdEnv.get("PATH")
	suite.Require().NoError(err, "could not get PATH")
	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	suite.Assert().Contains(paths, ts.Dirs.Work, "Could not find installation path, output: %s", cp.TrimmedSnapshot())

	suite.FileExists(filepath.Join(ts.Dirs.Work, "state-svc.exe"))
	suite.FileExists(filepath.Join(ts.Dirs.Work, "state-tray.exe"))
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallPerl5_32DefaultWindows() {
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)
	suite.runInstallTestWindows("-c", "\"state activate ActiveState/Perl-5.32 --default\"")
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallPerl5_32_ActivateDefaultWindows() {
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)
	suite.runInstallTestWindows("-activate-default", "ActiveState/Perl-5.32")
}

func (suite *InstallScriptsIntegrationTestSuite) runInstallTest(installScriptArgs ...string) {
	if runtime.GOOS != "linux" {
		suite.T().SkipNow()
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T(), ts.Dirs.Work, false, true)

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

	computedCommand := append([]string{script, "-t", ts.Dirs.Work, "-b", constants.BranchName}, installScriptArgs...)

	cp.ExpectExitCode(0)
	cp = ts.SpawnCmdWithOpts(
		"bash",
		e2e.WithArgs(computedCommand...),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false", "SHELL=bash"),
	)
	expectStateToolInstallation(cp)

	expectDefaultActivation(cp)
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

func (suite *InstallScriptsIntegrationTestSuite) runInstallTestWindows(installScriptArgs ...string) {
	if runtime.GOOS != "windows" {
		suite.T().SkipNow()
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	script := scriptPath(suite.T(), ts.Dirs.Work, false, true)

	isAdmin, err := osutils.IsWindowsAdmin()
	suite.Require().NoError(err, "Could not determine if running as administrator")

	cmdEnv := newCmdEnv(!isAdmin)
	oldPathEnv, err := cmdEnv.get("PATH")
	suite.Require().NoError(err, "could not get PATH")

	defer func() {
		err := cmdEnv.set("PATH", oldPathEnv)
		suite.Assert().NoError(err, "Unexpected error re-setting paths")
	}()

	computedCommand := append([]string{script, "-t", ts.Dirs.Work, "-b", constants.BranchName}, installScriptArgs...)

	cp := ts.SpawnCmdWithOpts(
		"powershell.exe",
		e2e.WithArgs(computedCommand...),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false", "SHELL="))
	expectStateToolInstallationWindows(cp)
	expectDefaultActivation(cp)
	cp.ExpectExitCode(0)

	pathEnv, err := cmdEnv.get("PATH")
	suite.Require().NoError(err, "could not get PATH")
	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	// The global binary directory is only added to the PATH for non-Administrator users
	if !isAdmin {
		suite.Assert().Contains(paths, filepath.Join(ts.Dirs.Cache, "bin"), "Could not find global binary directory on PATH")
	}
	suite.Assert().Contains(paths, ts.Dirs.Work, "Could not find installation path in PATH")
}

func (suite *InstallScriptsIntegrationTestSuite) BeforeTest(suiteName, testName string) {
	var err error
	root, err := environment.GetRootPath()
	suite.Require().NoError(err)
	testUpdateDir := filepath.Join(root, "build", "update")
	suite.Require().DirExists(testUpdateDir, "You need to run `state run generate-updates` for this test to work.")
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(testUpdateDir)))
	suite.server = &http.Server{Addr: "localhost:" + testPort, Handler: mux}
	go func() {
		_ = suite.server.ListenAndServe()
	}()

	suite.cfg, err = config.Get()
	suite.Require().NoError(err)
}

func (suite *InstallScriptsIntegrationTestSuite) AfterTest(suiteName, testName string) {
	err := suite.server.Shutdown(context.Background())
	suite.Require().NoError(err)
}

func TestInstallScriptsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InstallScriptsIntegrationTestSuite))
}
