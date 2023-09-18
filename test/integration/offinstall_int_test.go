package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/termtest"
	"github.com/mholt/archiver"

	"github.com/ActiveState/cli/internal/analytics/client/sync/reporters"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/offinstall"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/ActiveState/cli/internal/subshell/cmd"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/google/uuid"
)

type OffInstallIntegrationTestSuite struct {
	tagsuite.Suite

	installerPath   string
	uninstallerPath string
}

const (
	defaultOrg                 = "ActiveState-Test"
	defaultProject             = "IntegrationTest"
	anotherProject             = "Another-IntegrationTest"
	defaultArtifactsPayload    = "artifacts-payload"
	anotherArtifactsPayload    = "another-artifacts-payload"
	defaultInstalledExecutable = "test-offline-install"
	anotherInstalledExecutable = "test-another-offline-install"
)

func (suite *OffInstallIntegrationTestSuite) TestInstallAndUninstall() {
	suite.OnlyRunForTags(tagsuite.OffInstall)

	// Clean up env after test
	if runtime.GOOS == "windows" {
		env := cmd.NewCmdEnv(true)
		origPath, err := env.Get("PATH")
		suite.Require().NoError(err)
		defer func() {
			suite.Require().NoError(env.Set("PATH", origPath))
		}()
	} else {
		originalPath, exists := os.LookupEnv("PATH")
		defer func() {
			if !exists {
				return
			}
			suite.Require().NoError(os.Setenv("PATH", originalPath))
		}()
	}

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	testReportFilename := filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)
	suite.Require().NoFileExists(testReportFilename)

	fmt.Printf("Work dir: %s\n", ts.Dirs.Work)

	suite.preparePayload(ts, defaultArtifactsPayload, defaultProject)

	defaultInstallParentDir, err := offinstall.DefaultInstallParentDir()
	suite.Require().NoError(err)
	defaultInstallDir := filepath.Join(defaultInstallParentDir, "IntegrationTest")

	env := []string{constants.DisableRuntime + "=false"}
	if runtime.GOOS != "windows" {
		env = append(env, "SHELL=bash")
	}
	namespace := project.NewNamespace(defaultOrg, defaultProject, "")
	{ // Install
		suite.runOfflineInstaller(ts, defaultInstallDir, env)

		// Verify that our analytics event was fired
		time.Sleep(2 * time.Second) // give time to let rtwatcher detect process has exited
		events := parseAnalyticsEvents(suite, ts)
		suite.Require().NotEmpty(events)

		heartbeat := suite.filterEvent(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat)
		suite.assertDimensions(heartbeat)

		nDelete := countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeDelete, anaConst.SrcOfflineInstaller)
		if nDelete != 0 {
			suite.FailNow(fmt.Sprintf("Expected 0 delete events, got %d, events:\n%#v", nDelete, events))
		}

		// Ensure shell env is updated
		suite.assertShellUpdated(defaultInstallDir, namespace.String(), true, ts)

		// Ensure installation dir looks correct
		suite.assertInstallDir(defaultInstallDir, defaultInstalledExecutable, true)

		// Run executable and validate that it has the relocated value
		if runtime.GOOS == "windows" {
			refreshEnv := filepath.Join(environment.GetRootPathUnsafe(), "test", "integration", "testdata", "tools", "refreshenv", "refreshenv.bat")
			tp := ts.SpawnCmd("cmd", "/C", refreshEnv+" && "+defaultInstalledExecutable)
			tp.Expect("TEST REPLACEMENT", termtest.OptExpectTimeout(5*time.Second))
			tp.ExpectExitCode(0)
		} else {
			// Disabled for now: DX-1307
			// tp = ts.SpawnCmd("bash")
			// time.Sleep(1 * time.Second) // Give zsh a second to start -- can't use ExpectInput as it doesn't respect a custom HOME dir
			// tp.Send("test-offline-install")
			// tp.Expect("TEST REPLACEMENT", termtest.OptExpectTimeout(5*time.Second))
			// tp.Send("exit")
			// tp.ExpectExitCode(0)
		}
	}

	{ // Uninstall
		tp := ts.SpawnCmdWithOpts(
			suite.uninstallerPath,
			e2e.OptArgs(defaultInstallDir),
			e2e.OptAppendEnv(env...),
		)
		tp.Expect("continue?")
		tp.SendLine("y")
		tp.Expect("Uninstall Complete", termtest.OptExpectTimeout(5*time.Second))
		tp.Expect("Press enter to exit")
		tp.SendEnter()
		tp.ExpectExitCode(0)

		// Ensure shell env is updated
		suite.assertShellUpdated(defaultInstallDir, namespace.String(), false, ts)

		// Ensure installation files are removed
		suite.assertInstallDir(defaultInstallDir, defaultInstalledExecutable, false)

		// Verify that our analytics event was fired
		events := parseAnalyticsEvents(suite, ts)
		suite.Require().NotEmpty(events)
		nHeartbeat := countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat, anaConst.SrcExecutor)
		if nHeartbeat != 1 {
			suite.FailNow(fmt.Sprintf("Expected 1 heartbeat event, got %d, events:\n%#v", nHeartbeat, events))
		}
		del := suite.filterEvent(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeDelete)
		suite.assertDimensions(del)
	}
}

func (suite *OffInstallIntegrationTestSuite) TestInstallNoPermission() {
	suite.OnlyRunForTags(tagsuite.OffInstall)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	suite.preparePayload(ts, defaultArtifactsPayload, defaultProject)

	pathWithNoPermission := "/no-permission"
	if runtime.GOOS == "windows" {
		pathWithNoPermission = "C:\\Program Files\\No Permission"
	}

	tp := ts.SpawnCmdWithOpts(
		suite.installerPath,
		e2e.OptArgs(pathWithNoPermission),
	)
	tp.Expect("Please ensure that the directory is writeable", termtest.OptExpectTimeout(5*time.Second))
	tp.Expect("Press enter to exit", termtest.OptExpectTimeout(5*time.Second))
	tp.SendEnter()
	tp.ExpectExitCode(1)
}

func (suite *OffInstallIntegrationTestSuite) TestInstallMultiple() {
	suite.OnlyRunForTags(tagsuite.OffInstall)

	// Clean up env after test
	if runtime.GOOS == "windows" {
		env := cmd.NewCmdEnv(true)
		origPath, err := env.Get("PATH")
		suite.Require().NoError(err)
		defer func() {
			suite.Require().NoError(env.Set("PATH", origPath))
		}()
	} else {
		originalPath, exists := os.LookupEnv("PATH")
		defer func() {
			if !exists {
				return
			}
			suite.Require().NoError(os.Setenv("PATH", originalPath))
		}()
	}

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	testReportFilename := filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)
	suite.Require().NoFileExists(testReportFilename)

	suite.preparePayload(ts, defaultArtifactsPayload, defaultProject)

	defaultInstallParentDir, err := offinstall.DefaultInstallParentDir()
	suite.Require().NoError(err)
	firstInstallDir := filepath.Join(defaultInstallParentDir, "IntegrationTest")
	secondInstallDir := filepath.Join(defaultInstallParentDir, "Another-IntegrationTest")

	firstNamespace := project.NewNamespace(defaultOrg, defaultProject, "")
	secondNamespace := project.NewNamespace(defaultOrg, anotherProject, "")

	env := []string{constants.DisableRuntime + "=false"}
	if runtime.GOOS != "windows" {
		env = append(env, "SHELL=bash")
	}

	// Run offline installer for first project
	suite.runOfflineInstaller(ts, firstInstallDir, env)

	// Prepare new payload and run offline installer for second project
	suite.preparePayload(ts, anotherArtifactsPayload, anotherProject)
	suite.runOfflineInstaller(ts, secondInstallDir, env)

	// Assert first projects updates are still in place
	suite.assertShellUpdated(firstInstallDir, firstNamespace.String(), true, ts)
	suite.assertInstallDir(firstInstallDir, defaultInstalledExecutable, true)

	// Assert second projects updates are also in place
	suite.assertShellUpdated(secondInstallDir, firstNamespace.String(), true, ts)
	suite.assertInstallDir(secondInstallDir, anotherInstalledExecutable, true)

	// Uninstall first project
	suite.runOfflineUninstaller(ts, firstInstallDir, env)

	// Assert first project's update are removed
	suite.assertShellUpdated(firstInstallDir, firstNamespace.String(), false, ts)

	// Assert first project's installation files are removed
	suite.assertInstallDir(firstInstallDir, defaultInstalledExecutable, false)

	// Uninstall second project
	suite.runOfflineUninstaller(ts, secondInstallDir, env)

	// Assert second project's update are removed
	suite.assertShellUpdated(secondInstallDir, secondNamespace.String(), false, ts)

	// Assert second project's installation files are removed
	suite.assertInstallDir(secondInstallDir, anotherInstalledExecutable, false)
}

func (suite *OffInstallIntegrationTestSuite) TestInstallTwice() {
	suite.OnlyRunForTags(tagsuite.OffInstall)

	// Clean up env after test
	if runtime.GOOS == "windows" {
		env := cmd.NewCmdEnv(true)
		origPath, err := env.Get("PATH")
		suite.Require().NoError(err)
		defer func() {
			suite.Require().NoError(env.Set("PATH", origPath))
		}()
	} else {
		originalPath, exists := os.LookupEnv("PATH")
		defer func() {
			if !exists {
				return
			}
			suite.Require().NoError(os.Setenv("PATH", originalPath))
		}()
	}

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	suite.preparePayload(ts, defaultArtifactsPayload, defaultProject)

	defaultInstallParentDir, err := offinstall.DefaultInstallParentDir()
	suite.Require().NoError(err)
	defaultInstallDir := filepath.Join(defaultInstallParentDir, "IntegrationTest")

	env := []string{constants.DisableRuntime + "=false"}
	if runtime.GOOS != "windows" {
		env = append(env, "SHELL=bash")
	}

	suite.runOfflineInstaller(ts, defaultInstallDir, env)

	// Running offline installer again should not cause an error
	tp := ts.SpawnCmdWithOpts(
		suite.installerPath,
		e2e.OptArgs(defaultInstallDir),
		e2e.OptAppendEnv(env...),
	)
	tp.Expect("Installation directory is not empty")
	tp.Send("y")
	tp.Expect("Do you accept the ActiveState Runtime Installer License Agreement? (y/N)", termtest.OptExpectTimeout(5*time.Second))
	tp.Send("y")
	tp.Expect("Extracting", termtest.OptExpectTimeout(time.Second))
	tp.Expect("Installation complete")
	tp.Expect("Press enter to exit")
	tp.SendEnter()
	tp.ExpectExitCode(0)

	// Uninstall
	suite.runOfflineUninstaller(ts, defaultInstallDir, env)
}

func (suite *OffInstallIntegrationTestSuite) runOfflineInstaller(ts *e2e.Session, installDir string, env []string) {
	tp := ts.SpawnCmdWithOpts(
		suite.installerPath,
		e2e.OptArgs(installDir),
		e2e.OptAppendEnv(env...),
	)
	tp.Expect("Do you accept the ActiveState Runtime Installer License Agreement? (y/N)", termtest.OptExpectTimeout(5*time.Second))
	tp.Send("y")
	tp.Expect("Extracting", termtest.OptExpectTimeout(time.Second))
	tp.Expect("Installing")
	tp.Expect("Installation complete")
	tp.Expect("Press enter to exit")
	tp.SendEnter()
	tp.ExpectExitCode(0)
}

func (suite *OffInstallIntegrationTestSuite) runOfflineUninstaller(ts *e2e.Session, installDir string, env []string) {
	tp := ts.SpawnCmdWithOpts(
		suite.uninstallerPath,
		e2e.OptArgs(installDir),
		e2e.OptAppendEnv(env...),
	)
	tp.Expect("continue?")
	tp.SendLine("y")
	tp.Expect("Uninstall Complete", termtest.OptExpectTimeout(5*time.Second))
	tp.Expect("Press enter to exit")
	tp.SendEnter()
	tp.ExpectExitCode(0)
}

func (suite *OffInstallIntegrationTestSuite) preparePayload(ts *e2e.Session, payloadName, projectName string) {
	root := environment.GetRootPathUnsafe()

	suffix := "-windows"
	if runtime.GOOS != "windows" {
		suffix = "-nix"
	}

	// The payload is an artifact that contains mocked installation files
	payloadPath := filepath.Join(root, "test", "integration", "testdata", "offline-install", payloadName+suffix, "artifact")

	// The asset path contains additional files that we want to embed into the executable, such as the license
	assetPath := filepath.Join(root, "test", "integration", "testdata", "offline-install", "assets", projectName)

	// The payload archive is effectively double encrypted. We have the artifact itself, as well as the archive that
	// wraps it. Our test code only has one artifact, but in the wild there can and most likely will be multiple
	artifactMockPath := filepath.Join(ts.Dirs.Work, uuid.New().String()+".tar.gz")
	payloadMockPath := filepath.Join(ts.Dirs.Work, "artifacts.tar.gz")

	// The paths of the installer and uninstaller
	suite.installerPath = filepath.Join(ts.Dirs.Bin, "offline-installer"+exeutils.Extension)
	suite.uninstallerPath = filepath.Join(ts.Dirs.Bin, "uninstall"+exeutils.Extension)

	archiver := archiver.NewTarGz()
	{ // Create the artifact archive
		err := archiver.Archive(fileutils.ListFilesUnsafe(payloadPath), artifactMockPath)
		suite.Require().NoError(err)
	}

	{ // Create the payload archive which contains the artifact
		if fileutils.TargetExists(payloadMockPath) {
			err := os.RemoveAll(payloadMockPath)
			suite.Require().NoError(err)
		}
		err := archiver.Archive([]string{artifactMockPath}, payloadMockPath)
		suite.Require().NoError(err)
	}

	{ // Use a distinct copy of the installer to test with
		err := fileutils.CopyFile(filepath.Join(root, "build", "offline", "offline-installer"+exeutils.Extension), suite.installerPath)
		suite.Require().NoError(err)
	}

	{ // Use a distinct copy of the uninstaller to test with
		err := fileutils.CopyFile(filepath.Join(root, "build", "offline", "uninstall"+exeutils.Extension), suite.uninstallerPath)
		suite.Require().NoError(err)
	}

	// Copy all assets to same dir so gozip doesn't include their relative or absolute paths
	buildPath := filepath.Join(ts.Dirs.Work, "build")
	suite.Require().NoError(fileutils.MkdirUnlessExists(buildPath))
	suite.Require().NoError(fileutils.CopyMultipleFiles(map[string]string{
		payloadMockPath: filepath.Join(buildPath, filepath.Base(payloadMockPath)),
		filepath.Join(assetPath, "installer_config.json"): filepath.Join(buildPath, "installer_config.json"),
		filepath.Join(assetPath, "LICENSE.txt"):           filepath.Join(buildPath, "LICENSE.txt"),
		suite.uninstallerPath:                             filepath.Join(buildPath, filepath.Base(suite.uninstallerPath)),
	}))

	// Append our assets to the installer executable
	tp := ts.SpawnCmdWithOpts("gozip",
		e2e.OptWD(buildPath),
		e2e.OptArgs(
			"-c", suite.installerPath,
			filepath.Base(payloadMockPath),
			"installer_config.json",
			"LICENSE.txt",
			filepath.Base(suite.uninstallerPath),
		),
	)
	tp.ExpectExitCode(0)

	suite.Require().NoError(os.Chmod(suite.installerPath, 0775))   // ensure file is executable
	suite.Require().NoError(os.Chmod(suite.uninstallerPath, 0775)) // ensure file is executable
}

func (suite *OffInstallIntegrationTestSuite) assertShellUpdated(dir, namespace string, exists bool, ts *e2e.Session) {
	if runtime.GOOS != "windows" {
		// Test bashrc
		homeDir, err := user.HomeDir()
		suite.Require().NoError(err)

		fname := ".bashrc"
		if runtime.GOOS == "darwin" {
			fname = ".bash_profile"
		}

		assert := suite.Contains
		if !exists {
			assert = suite.NotContains
		}

		fpath := filepath.Join(homeDir, fname)
		rcContents := fileutils.ReadFileUnsafe(fpath)
		assert(string(rcContents), fmt.Sprintf("%s-%s", constants.RCAppendOfflineInstallStartLine, namespace), fpath)
		assert(string(rcContents), fmt.Sprintf("%s-%s", constants.RCAppendOfflineInstallStopLine, namespace), fpath)
		assert(string(rcContents), dir)
	} else {
		// It seems there is a race condition with updating the registry and asserting it was updated
		time.Sleep(time.Second)

		// Test registry
		isAdmin, err := osutils.IsAdmin()
		suite.Require().NoError(err)
		regKey := `HKCU\Environment`
		if isAdmin {
			regKey = `HKLM\SYSTEM\ControlSet001\Control\Session Manager\Environment`
		}
		out, err := exec.Command("reg", "query", regKey, "/v", "Path").Output()
		suite.Require().NoError(err)

		assert := strings.Contains
		if !exists {
			assert = func(s, substr string) bool {
				return !strings.Contains(s, substr)
			}
		}

		// we need to look for the short and the long version of the target PATH, because Windows translates between them arbitrarily
		shortPath, _ := fileutils.GetShortPathName(dir)
		longPath, _ := fileutils.GetLongPathName(dir)
		if !assert(string(out), shortPath) && !assert(string(out), longPath) && !assert(string(out), dir) {
			suite.T().Errorf("registry PATH \"%s\" validation failed for \"%s\", \"%s\" or \"%s\", should contain: %v", out, dir, shortPath, longPath, exists)
		}
	}
}

func (suite *OffInstallIntegrationTestSuite) filterEvent(events []reporters.TestLogEntry, category string, action string) reporters.TestLogEntry {
	ev := filterEvents(events, func(e reporters.TestLogEntry) bool {
		return e.Category == category && e.Action == action
	})
	suite.Require().Len(ev, 1)
	return ev[0]
}

func (suite *OffInstallIntegrationTestSuite) assertInstallDir(dir, executable string, exists bool) {
	assert := suite.Require().FileExists
	if !exists {
		assert = suite.Require().NoFileExists
	}
	if runtime.GOOS == "windows" {
		assert(filepath.Join(dir, "bin", fmt.Sprintf("%s.bat", executable)))
	} else {
		assert(filepath.Join(dir, "bin", fmt.Sprintf("%s", executable)))
	}
	if runtime.GOOS == "windows" {
		assert(filepath.Join(dir, "bin", "shell.bat"))
	}
	assert(filepath.Join(dir, "LICENSE.txt"))
}

func (suite *OffInstallIntegrationTestSuite) assertDimensions(event reporters.TestLogEntry) {
	evdbg, err := json.Marshal(event)
	suite.Require().NoError(err)
	dbg := fmt.Sprintf("Event: %s", string(evdbg))
	suite.Require().NotNil(event.Dimensions.ProjectNameSpace, dbg)
	suite.Require().NotNil(event.Dimensions.CommitID, dbg)
	suite.Require().Equal("ActiveState-Test/IntegrationTest", *event.Dimensions.ProjectNameSpace)
	suite.Require().Equal("00000000-0000-0000-0000-000000000000", *event.Dimensions.CommitID)
}

func TestOffInstallIntegrationTestSuite(t *testing.T) {
	t.Skip("Skipping offline installer tests as they will soon live in a separate repo")
	// suite.Run(t, new(OffInstallIntegrationTestSuite))
}
