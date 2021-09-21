package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/internal/testhelpers/updateinfomock"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"
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
	ext := ".ps1"
	if runtime.GOOS != "windows" {
		ext = ".sh"
	}
	name := "install" + ext
	if legacy {
		name = "legacy-install" + ext
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
		b = bytes.Replace(b, []byte(constants.APIUpdateInfoURL), []byte("http://localhost:"+updateinfomock.TestPort), -1)
		require.Contains(t, string(b), "http://localhost:"+updateinfomock.TestPort)
		b = bytes.Replace(b, []byte(constants.APIUpdateURL), []byte("http://localhost:"+updateinfomock.TestPort), -1)
	}

	scriptPath := filepath.Join(targetDir, filepath.Base(exec))
	err = ioutil.WriteFile(scriptPath, b, 0775)
	require.NoError(t, err)

	return scriptPath
}

type InstallScriptsIntegrationTestSuite struct {
	tagsuite.Suite
}

func expectLegacyStateToolInstallation(cp *termtest.ConsoleProcess, addToPathAnswer string) {
	cp.Expect("Installing to")
	cp.Expect("proceed with install?")
	cp.SendLine("Y")
	cp.Expect("Fetching version info")
	cp.Expect("Allow $PATH to be appended in your", 20*time.Second)
	cp.SendLine(addToPathAnswer)
	cp.Expect("State Tool installation complete")
}

func expectStateToolInstallation(cp *termtest.ConsoleProcess) {
	cp.Expect("proceed with install?")
	cp.SendLine("Y")
	cp.Expect("Fetching the latest version")
	cp.Expect("State Tool successfully installed.", time.Second*20)
}

func expectVersionedStateToolInstallation(cp *termtest.ConsoleProcess, version string) {
	cp.Expect("proceed with install?")
	cp.SendLine("Y")
	cp.Expect(fmt.Sprintf("Fetching version: %s", version))
	cp.Expect("State Tool successfully installed.")
}

func expectStateToolInstallationWindows(cp *termtest.ConsoleProcess) {
	cp.Expect("proceed with install?")
	cp.SendLine("Y")
	cp.Expect("Fetching the latest version")
	cp.Expect("State Tool successfully installed.")
}

func expectVersionedStateToolInstallationWindows(cp *termtest.ConsoleProcess, version string) {
	cp.Expect("proceed with install?")
	cp.SendLine("Y")
	cp.Expect(fmt.Sprintf("Fetching version: %s", version))
	cp.Expect("Installation Complete")
}

func expectLegacyStateToolInstallationWindows(cp *termtest.ConsoleProcess) {
	cp.Expect("Installing to")
	cp.Expect("proceed with install?")
	cp.SendLine("Y")
	cp.Expect("Fetching version info")
	cp.Expect("State Tool successfully installed to")
}

func expectDefaultActivation(cp *termtest.ConsoleProcess) {
	cp.Expect("Activating Virtual Environment")
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
	cp.Expect("Please provide an argument for parameter '-v'")
	cp.ExpectExitCode(1)

	cp = ts.SpawnCmdWithOpts("bash", e2e.WithArgs(script, "-t", ts.Dirs.Work, "-v", oldReleaseUpdateVersion))
	expectLegacyStateToolInstallation(cp, "n")
	cp.Expect("State Tool Installed")
	cp.ExpectExitCode(0)
}

// TestLegacyInstallShInstallMultiFileUpdate is meant to test whether installing via the legacy installer results in
// a multi-file state tool. This is achieved using the state-transition-update
func (suite *InstallScriptsIntegrationTestSuite) TestLegacyInstallShInstallMultiFileUpdate() {
	if runtime.GOOS == "windows" {
		suite.T().SkipNow()
	}

	tagName := "experiment"
	server := suite.setupMockServer()
	server.SetLegacyUpdateModifier(func(up *updateinfomock.LegacyInfo, _ string, _ string) {
		up.Tag = tagName
	})
	server.SetUpdateModifier(func(up *updater.AvailableUpdate, _ string, tag string) {
		if tag != tagName {
			return
		}
		up.Tag = &tagName
	})
	defer server.Close()

	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	script := scriptPath(suite.T(), ts.Dirs.Work, true, true)

	cp := ts.SpawnCmdWithOpts(
		"bash",
		e2e.WithArgs(script, "-t", ts.Dirs.Work, "-b", constants.BranchName, "-v", constants.Version),
		e2e.AppendEnv(updateinfomock.MockedUpdateServerEnvVars()...),
		e2e.AppendEnv(
			fmt.Sprintf("%s=%s", constants.OverwriteDefaultInstallationPathEnvVarName, filepath.Join(ts.Dirs.Work, "multi-file")),
		))

	expectLegacyStateToolInstallation(cp, "n")
	cp.Expect("State Tool Installed")
	cp.ExpectExitCode(0)

	// The transitional state tool should stay around and forward to the new installation
	suite.FileExists(filepath.Join(ts.Dirs.Work, "state"))
	// Note: When updating from an old update, we always install to the default installation path.
	// The default installation path is set to <ts.Dirs.Work>/multi-file for this test.
	suite.FileExists(filepath.Join(ts.Dirs.Work, "multi-file", "state-svc"))
	suite.FileExists(filepath.Join(ts.Dirs.Work, "multi-file", "state-tray"))

	// ensure that tagName is forwarded and stored in database
	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	defer cfg.Close()
	suite.Assert().Equal(tagName, cfg.GetString(updater.CfgUpdateTag))

	// after pulling in the multi-file update, we expect up to TWO more requests to the update server: for the auto-update check (possibly), and the `state update` request
	server.ExpectAtLeastNRequests(2)
	server.NthRequest(0).ExpectQueryParam("source", "install")
	server.NthRequest(0).ExpectLegacyQuery(true)
	server.NthRequest(0).ExpectTagResponse(&tagName)
	server.NthRequest(1).ExpectQueryParam("source", "update")
	server.NthRequest(1).ExpectQueryParam("tag", tagName)
	server.NthRequest(1).ExpectLegacyQuery(false)
	server.NthRequest(1).ExpectTagResponse(&tagName)

	cp = ts.SpawnCmd(filepath.Join(ts.Dirs.Work, "state"), "clean", "uninstall")
	cp.Expect("You are about to remove")
	cp.SendLine("y")
	cp.ExpectExitCode(0)

	// Ensure that transitional State Tool has been removed
	suite.NoFileExists(filepath.Join(ts.Dirs.Work, "state"))
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallSh() {
	if runtime.GOOS == "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.InstallScripts, tagsuite.Critical)
	tagName := "experiment"

	tests := []struct {
		Name        string
		TestInstall bool
		Tag         *string
		Channel     string
	}{
		{"install-local-test-update", true, nil, constants.BranchName},
		{"install-local-test-update-with-tag", true, &tagName, constants.BranchName},
		// Todo https://www.pivotaltracker.com/story/show/177863116
		// Replace the target branch for this test to release, as soon as we have a working deployment there.
		{"install-release", false, nil, "master"},
	}

	for _, tt := range tests {
		suite.Run(tt.Name, func() {
			server := suite.setupMockServer()
			server.SetUpdateModifier(func(up *updater.AvailableUpdate, source, tag string) {
				if source != "install" {
					return
				}

				// If the update is tagged (which shouldn't happen), respond with an invalid version, so we can test that the tag name was forwarded to the server
				if tag == "experiment" {
					up.Version = "99.99.99"
					up.Path = "invalid-path"
					return
				}

				// set the tag
				up.Tag = tt.Tag
			})
			defer server.Close()
			dir, err := installation.LauncherInstallPath()
			suite.Require().NoError(err)
			var extraEnv []string
			if runtime.GOOS == "linux" {
				dir, err = ioutil.TempDir("", "temp_home*")
				suite.Require().NoError(err)
				extraEnv = append(extraEnv, fmt.Sprintf("HOME=%s", dir), fmt.Sprintf("_TEST_SYSTEM_PATH=%s", dir))
			}

			ts := e2e.New(suite.T(), false, extraEnv...)
			defer ts.Close()

			script := scriptPath(suite.T(), ts.Dirs.Work, false, tt.TestInstall)

			cp := ts.SpawnCmdWithOpts("bash", e2e.WithArgs(script, "-t", ts.Dirs.Work, "-b", tt.Channel))
			expectStateToolInstallation(cp)
			cp.Expect("State Tool Installed")
			cp.ExpectExitCode(0)

			assertApplicationDirContents(suite.Contains, dir)
			assertBinDirContents(suite.Contains, ts.Dirs.Work)
			suite.DirExists(ts.Dirs.Config)

			// Only test the un-installation on local update (Review once installed updates become more stable)
			if !tt.TestInstall {
				return
			}
			// Check that the tag is set
			cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
			suite.Require().NoError(err)
			defer cfg.Close()
			var tag *string
			if ctag := cfg.GetString(updater.CfgUpdateTag); ctag != "" {
				tag = &ctag
			}
			suite.Assert().Equal(tt.Tag, tag)

			cp = ts.SpawnCmdWithOpts(filepath.Join(ts.Dirs.Work, "state"+osutils.ExeExt), e2e.WithArgs("clean", "uninstall"))
			cp.Expect("Please Confirm")
			cp.SendLine("y")
			cp.ExpectExitCode(0)

			assertApplicationDirContents(suite.NotContains, dir)
			assertBinDirContents(suite.NotContains, ts.Dirs.Work)

			// We expect up to two requests: one from the install script, and potentially another one for the initial auto-update check in the state-svc
			server.ExpectAtLeastNRequests(1)
			server.NthRequest(0).ExpectQueryParam("source", "install")
			server.NthRequest(0).ExpectTagResponse(tt.Tag)
		})
	}
}

// assertApplicationDirContents checks if given files are or are not in the application directory
func assertApplicationDirContents(assertFunc func(s, c interface{}, msg ...interface{}) bool, dir string) {
	homeDirFiles := listFilesOnly(dir)
	switch runtime.GOOS {
	case "linux":
		assertFunc(homeDirFiles, "state-tray.desktop")
		assertFunc(homeDirFiles, "state-tray.svg")
	case "darwin":
		assertFunc(homeDirFiles, "Info.plist")
		assertFunc(homeDirFiles, "state-tray.icns")
	case "windows":
		assertFunc(homeDirFiles, "state-tray.lnk")
		assertFunc(homeDirFiles, "state-tray.icns")
	}
}

// assertBinDirContents checks if given files are or are not in the bin directory
func assertBinDirContents(assertFunc func(s, c interface{}, msg ...interface{}) bool, dir string) {
	binFiles := listFilesOnly(dir)
	assertFunc(binFiles, "state"+osutils.ExeExt)
	assertFunc(binFiles, "state-tray"+osutils.ExeExt)
	assertFunc(binFiles, "state-svc"+osutils.ExeExt)
}

// listFilesOnly is a helper function for assertBinDirContents and assertApplicationDirContents filtering a directory recursively for base filenames
// It allows for simple and coarse checks if a file exists or does not exist in the directory structure
func listFilesOnly(dir string) []string {
	files := fileutils.ListDir(dir, true)
	files = funk.Filter(files, func(f string) bool {
		return !fileutils.IsDir(f)
	}).([]string)
	return funk.Map(files, filepath.Base).([]string)
}

func (suite *InstallScriptsIntegrationTestSuite) TestInstallShVersion() {
	if runtime.GOOS == "windows" {
		suite.T().SkipNow()
	}
	suite.OnlyRunForTags(tagsuite.InstallScripts)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	expected := "0.29.0-SHAb58b472"
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
	tagName := "experiment"

	tests := []struct {
		Name        string
		TestInstall bool
		Tag         *string
		Channel     string
	}{
		{"install-local-test-update", true, nil, constants.BranchName},
		{"install-local-test-update-with-tag", true, &tagName, constants.BranchName},
		// Todo https://www.pivotaltracker.com/story/show/177863116
		// Replace the target branch for this test to release, as soon as we have a working deployment there.
		{"install-release", false, nil, "master"},
	}

	for _, tt := range tests {
		suite.Run(tt.Name, func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			server := suite.setupMockServer()
			server.SetUpdateModifier(func(up *updater.AvailableUpdate, source, tag string) {
				if source != "install" {
					return
				}

				// If the update is tagged (which shouldn't happen), respond with an invalid version, so we can test that the tag name was forwarded to the server
				if tag == "experiment" {
					up.Version = "99.99.99"
					up.Path = "invalid-path"
					return
				}

				// set the tag
				up.Tag = tt.Tag
			})
			defer server.Close()

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

			assertBinDirContents(suite.Contains, ts.Dirs.Work)

			pathEnv, err := cmdEnv.get("PATH")
			suite.Require().NoError(err, "could not get PATH")
			paths := strings.Split(pathEnv, string(os.PathListSeparator))
			suite.Assert().Contains(paths, ts.Dirs.Work, "Could not find installation path, output: %s", cp.TrimmedSnapshot())

			// Only test the un-installation on local update (Review once installed updates become more stable)
			if !tt.TestInstall {
				return
			}
			// give some time for the provided state-tray app to start and write its pid to the config file
			time.Sleep(time.Second)

			// Check that the tag is set
			cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
			suite.Require().NoError(err)
			defer cfg.Close()
			var tag *string
			ct := cfg.GetString(updater.CfgUpdateTag)
			if ct != "" {
				tag = &ct
			}
			suite.Assert().Equal(tt.Tag, tag)

			cp = ts.SpawnCmdWithOpts(filepath.Join(ts.Dirs.Work, "state"+osutils.ExeExt), e2e.WithArgs("clean", "uninstall"))
			cp.Expect("Please Confirm")
			cp.SendLine("y")
			cp.ExpectExitCode(0)

			// wait three seconds until state.exe is removed (in the background)
			time.Sleep(time.Second * 4)

			// Todo: Sometimes the state.exe file still remains on disk (always on CI, never on my machine!)
			// https://www.pivotaltracker.com/story/show/178148949
			// assertBinDirContents(suite.NotContains, ts.Dirs.Work)

			// Todo: Remove the following lines if the bug above is fixed
			binFiles := listFilesOnly(ts.Dirs.Work)
			suite.NotContains(binFiles, "state-tray"+osutils.ExeExt)
			suite.NotContains(binFiles, "state-svc"+osutils.ExeExt)

			// We expect up to two requests: one from the install script, and potentially another one for the initial auto-update check
			server.ExpectAtLeastNRequests(1)
			server.NthRequest(0).ExpectQueryParam("source", "install")
			server.NthRequest(0).ExpectTagResponse(tt.Tag)
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

	expected := "0.29.0-SHAb58b472"
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

	expected := "0.29.0-SHAb58b472"
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
	cp.ExpectLongString("Please provide an argument for parameter '-v'")
	cp.ExpectExitCode(1)

	cp = ts.SpawnCmdWithOpts("powershell.exe", e2e.WithArgs(script, "-t", ts.Dirs.Work, "-v", oldReleaseUpdateVersion), e2e.AppendEnv("SHELL="))
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

	tagName := "experiment"
	server := suite.setupMockServer()
	server.SetLegacyUpdateModifier(func(up *updateinfomock.LegacyInfo, _ string, _ string) {
		up.Tag = tagName
	})
	server.SetUpdateModifier(func(up *updater.AvailableUpdate, _ string, tag string) {
		if tag != tagName {
			return
		}
		up.Tag = &tagName
	})
	defer server.Close()

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
		e2e.WithArgs(script, "-t", ts.Dirs.Work, "-b", constants.BranchName, "-v", constants.Version),
		e2e.AppendEnv(updateinfomock.MockedUpdateServerEnvVars()...),
		e2e.AppendEnv(
			"SHELL=",
			fmt.Sprintf("%s=%s", constants.OverwriteDefaultInstallationPathEnvVarName, filepath.Join(ts.Dirs.Work, "multi-file")),
		))

	expectLegacyStateToolInstallationWindows(cp)
	cp.ExpectExitCode(0)

	pathEnv, err := cmdEnv.get("PATH")
	suite.Require().NoError(err, "could not get PATH")
	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	suite.Assert().Contains(paths, ts.Dirs.Work, "Could not find installation path, output: %s", cp.TrimmedSnapshot())

	// The transitional state tool should be kept around and forward to the new default installation
	suite.FileExists(filepath.Join(ts.Dirs.Work, "state.exe"))
	// Note: When updating from an old update, we always install to the default installation path.
	// The default installation path is set to <ts.Dirs.Work>/multi-file for this test.
	suite.FileExists(filepath.Join(ts.Dirs.Work, "multi-file", "state.exe"))
	suite.FileExists(filepath.Join(ts.Dirs.Work, "multi-file", "state-svc.exe"))
	suite.FileExists(filepath.Join(ts.Dirs.Work, "multi-file", "state-tray.exe"))

	// ensure that tagName is forwarded and stored in database
	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	defer cfg.Close()
	suite.Assert().Equal(tagName, cfg.GetString(updater.CfgUpdateTag))

	// We expect two more requests after an update to the multi-file State Tool: possibly one for the initial auto-update check, and another one for the `state update` call
	server.ExpectAtLeastNRequests(2)
	server.NthRequest(0).ExpectLegacyQuery(true)
	server.NthRequest(0).ExpectQueryParam("source", "install")
	server.NthRequest(0).ExpectTagResponse(&tagName)
	server.NthRequest(1).ExpectLegacyQuery(false)
	server.NthRequest(1).ExpectQueryParam("source", "update")
	server.NthRequest(1).ExpectTagResponse(&tagName)

	cp = ts.SpawnCmd(filepath.Join(ts.Dirs.Work, "state.exe"), "clean", "uninstall")
	cp.Expect("You are about to remove")
	cp.SendLine("y")
	cp.ExpectExitCode(0)

	time.Sleep(500 * time.Millisecond)

	// Ensure that transitional State Tool has been removed
	suite.NoFileExists(filepath.Join(ts.Dirs.Work, "state"))
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

	server := suite.setupMockServer()
	defer server.Close()

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

	// We expect up two requests: one from the install script, and possibly another one for the initial auto-update check in the state-svc
	server.ExpectAtLeastNRequests(1)
	server.NthRequest(0).ExpectQueryParam("source", "install")
	server.NthRequest(0).ExpectTagResponse(nil)
	server.NthRequest(0).ExpectLegacyQuery(false)
}

func (suite *InstallScriptsIntegrationTestSuite) runInstallTestWindows(installScriptArgs ...string) {
	if runtime.GOOS != "windows" {
		suite.T().SkipNow()
	}

	server := suite.setupMockServer()
	defer server.Close()

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

	// We expect up to two requests: one from the install script, and possibly another one for the initial auto-update check in the state-svc
	server.ExpectAtLeastNRequests(1)
	server.NthRequest(0).ExpectLegacyQuery(false)
	server.NthRequest(0).ExpectQueryParam("source", "install")
	server.NthRequest(0).ExpectTagResponse(nil)
}

func (suite *InstallScriptsIntegrationTestSuite) setupMockServer() *updateinfomock.MockUpdateInfoServer {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err)
	testUpdateDir := filepath.Join(root, "build", "update")
	suite.Require().DirExists(testUpdateDir, "You need to run `state run generate-updates` for this test to work.")

	return updateinfomock.New(suite.Suite.Suite, testUpdateDir)
}

func TestInstallScriptsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InstallScriptsIntegrationTestSuite))
}
