package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/archiver"
	"github.com/ActiveState/cli/internal/analytics/client/sync/reporters"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/subshell/cmd"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
)

type OffInstallIntegrationTestSuite struct {
	tagsuite.Suite

	installerPath   string
	uninstallerPath string
}

func (suite *OffInstallIntegrationTestSuite) TestInstallAndUninstall() {
	suite.OnlyRunForTags(tagsuite.OffInstall)

	// Clean up env after test (windows only for now)
	if runtime.GOOS == "windows" {
		env := cmd.NewCmdEnv(true)
		origPath, err := env.Get("PATH")
		suite.Require().NoError(err)
		defer func() {
			suite.Require().NoError(env.Set("PATH", origPath))
		}()
	}

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	testReportFilename := filepath.Join(ts.Dirs.Config, reporters.TestReportFilename)
	suite.Require().NoFileExists(testReportFilename)

	fmt.Printf("Work dir: %s\n", ts.Dirs.Work)

	suite.preparePayload(ts)
	targetDir := filepath.Join(ts.Dirs.Work, "target")

	env := []string{
		"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
		"VERBOSE=true",
	}
	if runtime.GOOS != "windows" {
		env = append(env, "SHELL=zsh")
	}
	{ // Install
		tp := ts.SpawnCmdWithOpts(
			suite.installerPath,
			e2e.WithArgs(targetDir),
			e2e.AppendEnv(env...),
		)
		// tp.Expect("Stage 1 of 3 Finished")
		tp.Expect("Do you accept the ActiveState License")
		tp.SendLine("")
		// tp.Expect("Stage 2 of 3 Finished")
		// tp.Expect("Stage 3 of 3 Finished")
		tp.Expect("Setup environment for installed project?")
		tp.Send("Y")
		tp.ExpectExitCode(0)
		fmt.Println(tp.Snapshot())

		// Verify that our analytics event was fired
		time.Sleep(2 * time.Second) // give time to let rtwatcher detect process has exited
		events := parseAnalyticsEvents(suite, ts)
		suite.Require().NotEmpty(events)
		nHeartbeat := countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat)
		if nHeartbeat != 1 {
			suite.FailNow(fmt.Sprintf("Expected 1 heartbeat event, got %d, events:\n%#v", nHeartbeat, events))
		}
		nDelete := countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeDelete)
		if nDelete != 0 {
			suite.FailNow(fmt.Sprintf("Expected 0 delete events, got %d, events:\n%#v", nDelete, events))
		}

		// Ensure shell env is updated
		suite.assertShellUpdated(targetDir, true, ts)

		// Ensure installation dir looks correct
		suite.assertInstallDir(targetDir, true)

		// Run executable and validate that it has the relocated value
		if runtime.GOOS == "windows" {
			refreshEnv := filepath.Join(environment.GetRootPathUnsafe(), "test", "integration", "testdata", "tools", "refreshenv", "refreshenv.bat")
			tp = ts.SpawnCmd("cmd", "/C", refreshEnv+" && test-offline-install")
		} else {
			tp = ts.SpawnCmd("zsh")
			tp.WaitForInput(time.Second * 5)
			tp.Send("test-offline-install")
			tp.Send("exit")
		}
		tp.Expect("TEST REPLACEMENT", 5*time.Second)
		tp.ExpectExitCode(0)
	}

	{ // Uninstall
		tp := ts.SpawnCmdWithOpts(
			suite.uninstallerPath,
			e2e.WithArgs(targetDir),
			e2e.AppendEnv(env...),
		)
		tp.Expect("Uninstall Complete", 5*time.Second)
		tp.ExpectExitCode(0)

		// Ensure shell env is updated
		suite.assertShellUpdated(targetDir, false, ts)

		// Ensure installation files are removed
		suite.assertInstallDir(targetDir, false)

		// Verify that our analytics event was fired
		events := parseAnalyticsEvents(suite, ts)
		suite.Require().NotEmpty(events)
		nHeartbeat := countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeHeartbeat)
		if nHeartbeat != 1 {
			suite.FailNow(fmt.Sprintf("Expected 1 heartbeat event, got %d, events:\n%#v", nHeartbeat, events))
		}
		nDelete := countEvents(events, anaConst.CatRuntimeUsage, anaConst.ActRuntimeDelete)
		if nDelete != 1 {
			suite.FailNow(fmt.Sprintf("Expected 1 delete events, got %d, events:\n%#v", nDelete, events))
		}
	}
}

func (suite *OffInstallIntegrationTestSuite) preparePayload(ts *e2e.Session) {
	root := environment.GetRootPathUnsafe()

	suffix := "-windows"
	if runtime.GOOS != "windows" {
		suffix = "-nix"
	}

	// The payload is an artifact that contains mocked installation files
	payloadPath := filepath.Join(root, "test", "integration", "testdata", "offline-install", "artifacts-payload"+suffix, "artifact")

	// The asset path contains additional files that we want to embed into the executable, such as the license
	assetPath := filepath.Join(root, "test", "integration", "testdata", "offline-install", "assets")

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
		e2e.WithWorkDirectory(buildPath),
		e2e.WithArgs(
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

func (suite *OffInstallIntegrationTestSuite) assertShellUpdated(dir string, exists bool, ts *e2e.Session) {
	if runtime.GOOS != "windows" {
		// Test zshrc
		homeDir, err := os.UserHomeDir()
		suite.Require().NoError(err)

		fname := ".zshrc"

		assert := suite.Contains
		if !exists {
			assert = suite.NotContains
		}

		fpath := filepath.Join(homeDir, fname)
		rcContents := fileutils.ReadFileUnsafe(fpath)
		assert(string(rcContents), constants.RCAppendOfflineInstallStartLine, fpath)
		assert(string(rcContents), constants.RCAppendOfflineInstallStopLine, fpath)
		assert(string(rcContents), dir)
	} else {
		// Test registry
		out, err := exec.Command("reg", "query", `HKEY_CURRENT_USER\Environment`, "/v", "Path").Output()
		suite.Require().NoError(err)

		assert := strings.Contains
		if !exists {
			assert = func(s, substr string) bool {
				return !strings.Contains(s, substr)
			}
		}

		// we need to look for  the short and the long version of the target PATH, because Windows translates between them arbitrarily
		shortPath, _ := fileutils.GetShortPathName(dir)
		longPath, _ := fileutils.GetLongPathName(dir)
		if !assert(string(out), shortPath) && !assert(string(out), longPath) && !assert(string(out), dir) {
			suite.T().Errorf("registry PATH \"%s\" validation failed for \"%s\", \"%s\" or \"%s\", should contain: %v", out, dir, shortPath, longPath, exists)
		}
	}
}

func (suite *OffInstallIntegrationTestSuite) assertInstallDir(dir string, exists bool) {
	assert := suite.Require().FileExists
	if !exists {
		assert = suite.Require().NoFileExists
	}
	if runtime.GOOS == "windows" {
		assert(filepath.Join(dir, "bin", "test-offline-install.bat"))
	} else {
		assert(filepath.Join(dir, "bin", "test-offline-install"))
	}
	if runtime.GOOS == "windows" {
		assert(filepath.Join(dir, "bin", "shell.bat"))
	}
	assert(filepath.Join(dir, "LICENSE.txt"))
}

func TestOffInstallIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(OffInstallIntegrationTestSuite))
}
