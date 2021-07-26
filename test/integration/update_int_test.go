package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type UpdateIntegrationTestSuite struct {
	tagsuite.Suite
	cfg    projectfile.ConfigGetter
	server *http.Server
}

type matcherFunc func(expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool

// Todo https://www.pivotaltracker.com/story/show/177863116
// Update to release branch when possible
var targetBranch = "beta"
var testBranch = "test-channel"
var oldUpdateVersion = "beta@0.28.1-SHA8592c6a"
var oldReleaseUpdateVersion = "0.28.2-SHAbdac00e"
var specificVersion = "0.29.0-SHA9f570a0"

func init() {
	if constants.BranchName == targetBranch {
		targetBranch = "master"
	}
}

var testPort = "24217"

func (suite *UpdateIntegrationTestSuite) BeforeTest(suiteName, testName string) {
	var err error
	root, err := environment.GetRootPath()
	suite.Require().NoError(err)
	testUpdateDir := filepath.Join(root, "build", "test-update")
	suite.Require().DirExists(testUpdateDir, "You need to run `state run generate-test-updates` for this test to work.")
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(testUpdateDir)))
	suite.server = &http.Server{Addr: "localhost:" + testPort, Handler: mux}
	go func() {
		_ = suite.server.ListenAndServe()
	}()

	suite.cfg, err = config.New()
	suite.Require().NoError(err)
}

func (suite *UpdateIntegrationTestSuite) AfterTest(suiteName, testName string) {
	err := suite.server.Shutdown(context.Background())
	suite.Require().NoError(err)
	suite.Require().NoError(suite.cfg.Close())
}

// env prepares environment variables for the test
// disableUpdates prevents all update code from running
// testUpdate directs to the locally running update directory and requires that a test update bundles has been generated with `state run generate-test-update`
func (suite *UpdateIntegrationTestSuite) env(disableUpdates, testUpdate bool) []string {
	env := []string{}

	if disableUpdates {
		env = append(env, "ACTIVESTATE_CLI_DISABLE_UPDATES=true")
	} else {
		env = append(env, "ACTIVESTATE_CLI_DISABLE_UPDATES=false")
	}

	if testUpdate {
		env = append(env, fmt.Sprintf("_TEST_UPDATE_URL=http://localhost:%s/", testPort))
	} else {
		env = append(env, fmt.Sprintf("%s=%s", constants.UpdateBranchEnvVarName, targetBranch))
	}

	dir, err := ioutil.TempDir("", "system*")
	suite.NoError(err)
	env = append(env, fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir))

	return env
}

func (suite *UpdateIntegrationTestSuite) versionCompare(ts *e2e.Session, disableUpdates, testUpdate bool, expected string, matcher matcherFunc) {
	type versionData struct {
		Version string `json:"version"`
	}

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExes()

	before := fileutils.ListDir(ts.Dirs.Config, false)

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"), e2e.AppendEnv(suite.env(disableUpdates, testUpdate)...))
	cp.ExpectExitCode(0)

	if !disableUpdates {
		// short timeout to wait for installation log file to be created
		time.Sleep(500 * time.Millisecond)
		after := fileutils.ListDir(ts.Dirs.Config, false)
		onlyAfter, _ := funk.Difference(after, before)
		logFile, ok := funk.FindString(onlyAfter.([]string), func(s string) bool { return strings.HasPrefix(filepath.Base(s), "state-installer") })
		if ok {
			suite.pollForUpdateFromLogfile(logFile)
			cp = ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"), e2e.AppendEnv(suite.env(disableUpdates, testUpdate)...))
			cp.ExpectExitCode(0)
		}
	}

	version := versionData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &version)

	matcher(expected, version.Version, fmt.Sprintf("Version could not be matched, output:\n\n%s", out))
}

func (suite *UpdateIntegrationTestSuite) branchCompare(ts *e2e.Session, disableUpdates bool, testUpdate bool, expected string, matcher matcherFunc) {
	type branchData struct {
		Branch string `json:"branch"`
	}

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExesLegacy()

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"), e2e.AppendEnv(suite.env(disableUpdates, testUpdate)...))
	cp.ExpectExitCode(0, 30*time.Second)

	branch := branchData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &branch)

	matcher(expected, branch.Branch, fmt.Sprintf("Branch could not be matched, output:\n\n%s", out))
}

func (suite *UpdateIntegrationTestSuite) TestUpdateAvailable() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	// use unique exe
	ts.UseDistinctStateExes()

	// Technically state tool automatically starts the state-svc, but the update notification only happens if the svc
	// happens to already be running and fails silently if not, so in this case we want to ensure the svc is running
	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false, true)...))
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(e2e.WithArgs("--version", "--verbose"))
	cp.Expect("Update Available")
	cp.ExpectExitCode(0)
}

func (suite *UpdateIntegrationTestSuite) pollForUpdateInBackground(configDir string, beforeFiles []string) string {
	for i := 0; i < 10; i++ {
		after := fileutils.ListDir(configDir, false)
		onlyAfter, _ := funk.Difference(after, beforeFiles)
		logFile, ok := funk.FindString(onlyAfter.([]string), func(s string) bool { return strings.HasPrefix(filepath.Base(s), "state-installer") })
		if ok {
			return suite.pollForUpdateFromLogfile(logFile)
		}
		time.Sleep(200 * time.Millisecond)
	}
	return ""
}

func (suite *UpdateIntegrationTestSuite) pollForUpdateFromLogfile(logFile string) string {
	var logs []byte
	// poll for successful auto-update
	for i := 0; i < 60; i++ {
		time.Sleep(time.Millisecond * 200)

		var err error
		logs, err = ioutil.ReadFile(logFile)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		suite.NoError(err)
		if strings.Contains(string(logs), "was successful") || strings.Contains(string(logs), "Installation failed") {
			return string(logs)
		}
	}

	if !fileutils.FileExists(logFile) {
		suite.T().Errorf("logFile does not exist: %s", logFile)
	} else {
		suite.T().Errorf("could not verify logFile contents at %s, contents:\n%s", logFile, string(logs))
	}

	return ""
}

func (suite *UpdateIntegrationTestSuite) TestUpdate() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	tests := []struct {
		Name                string
		TestUpdate          bool
		StateToolRunning    bool
		ExpectBackupCleaned bool
	}{
		{
			Name:                "test-update",
			TestUpdate:          true,
			StateToolRunning:    false,
			ExpectBackupCleaned: true,
		},
		{
			Name:                "actual-update",
			TestUpdate:          false,
			StateToolRunning:    false,
			ExpectBackupCleaned: true,
		},
		{
			Name:                "old-state-tool-running",
			TestUpdate:          true,
			StateToolRunning:    true,
			ExpectBackupCleaned: runtime.GOOS != "windows", // Note: On Windows we cannot remove the backup file when an old process is still running!
		},
	}
	for _, tt := range tests {
		if !tt.TestUpdate {
			// Todo https://www.pivotaltracker.com/story/show/177858645
			suite.T().Skip("This requires an update bundle to be released to the release branch")
		}
		suite.Run(tt.Name, func() {
			ts := e2e.New(suite.T(), true)
			defer ts.Close()

			suite.addProjectFileWithWaitingScript(ts.Dirs.Work)

			// Ensure we always use a unique exe for updates
			ts.UseDistinctStateExes()

			stopBg := make(chan struct{})
			var wg sync.WaitGroup

			if tt.StateToolRunning {
				wg.Add(1)
				go func() {
					defer wg.Done()
					bgCp := ts.SpawnWithOpts(e2e.WithArgs("run", "wait", "10"), e2e.AppendEnv(suite.env(false, tt.TestUpdate)...), e2e.BackgroundProcess())
					// need to close background process manually
					defer bgCp.Close()
					select {
					case <-stopBg:
					case <-time.After(time.Second * 10):
					}
					bgCp.Expect("Waiting for input")
					// interrupting the background process
					bgCp.SendLine("")
					bgCp.ExpectExitCode(0)
				}()
			}

			// Todo This should not be necessary https://www.pivotaltracker.com/story/show/177865635
			cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false, tt.TestUpdate)...))
			cp.ExpectExitCode(0)

			fakeHome := filepath.Join(ts.Dirs.Work, "home")
			err := fileutils.Mkdir(fakeHome)
			suite.Require().NoError(err)

			before := fileutils.ListDir(ts.Dirs.Config, false)

			cp = ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(suite.env(false, tt.TestUpdate)...), e2e.AppendEnv(fmt.Sprintf("HOME=%s", fakeHome)))
			cp.Expect("Updating State Tool to latest version available")
			cp.Expect(fmt.Sprintf("Version update to %s@", constants.BranchName))
			cp.ExpectExitCode(0)

			var logs string
			if tt.TestUpdate {
				fmt.Println(cp.TrimmedSnapshot())
				logs = suite.pollForUpdateInBackground(ts.Dirs.Config, before)
			}

			// tell background process to stop...
			close(stopBg)
			// ...and wait for it
			wg.Wait()

			if tt.TestUpdate {
				suite.Assert().Contains(logs, "was successful")
				if tt.ExpectBackupCleaned {
					suite.Assert().Contains(logs, "Removed all backup files.")
				} else {
					suite.Assert().Contains(logs, "Failed to remove backup files")
				}
			}
			suite.versionCompare(ts, true, tt.TestUpdate, constants.Version, suite.NotEqual)
		})
	}
}

func (suite *UpdateIntegrationTestSuite) TestUpdateChannel() {
	tests := []struct {
		Name       string
		TestUpdate bool
		Channel    string
		Version    string
	}{
		{"test-update", true, testBranch, ""},
		{"release-channel", false, targetBranch, ""},
		{"specific-update", false, targetBranch, specificVersion},
	}
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	for _, tt := range tests {
		suite.Run(tt.Name, func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			// Ensure we always use a unique exe for updates
			ts.UseDistinctStateExes()

			// Todo This should not be necessary https://www.pivotaltracker.com/story/show/177865635
			cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false, tt.TestUpdate)...))
			cp.ExpectExitCode(0)

			before := fileutils.ListDir(ts.Dirs.Config, false)

			info, err := os.Stat(ts.Exe)
			suite.Require().NoError(err)
			modTime := info.ModTime()

			updateArgs := []string{"update", "--set-channel", tt.Channel}
			if tt.Version != "" {
				updateArgs = append(updateArgs, "--set-version", tt.Version)
			}
			cp = ts.SpawnWithOpts(e2e.WithArgs(updateArgs...), e2e.AppendEnv(suite.env(false, tt.TestUpdate)...))
			if tt.Version == "" {
				cp.Expect("Updating State Tool to latest version available")
			} else {
				cp.Expect("Updating State Tool to version")
			}
			cp.Expect(fmt.Sprintf("Version update to %s@", tt.Channel))
			cp.ExpectExitCode(0)

			if tt.TestUpdate {
				logs := suite.pollForUpdateInBackground(ts.Dirs.Config, before)
				suite.Assert().Contains(logs, "was successful")
			} else {
				updated := false
				// wait for up to a minute for the State Tool to get modified
				for x := 0; x < 300; x++ {
					info, err := os.Stat(ts.Exe)
					if errors.Is(err, os.ErrNotExist) {
						continue
					}
					if !info.ModTime().Equal(modTime) {
						updated = true
						break
					}
					time.Sleep(200 * time.Millisecond)
				}
				suite.Require().True(updated, "Timeout: Expected the State Tool to get modified.")
			}

			// wait half a second for the State Tool to be written to disk completely
			time.Sleep(500 * time.Millisecond)

			suite.branchCompare(ts, false, tt.TestUpdate, tt.Channel, suite.Equal)

			if tt.Version != "" {
				suite.versionCompare(ts, true, false, tt.Version, suite.Equal)
			}
		})
	}
}

func (suite *UpdateIntegrationTestSuite) TestUpdateNoPermissions() {
	suite.OnlyRunForTags(tagsuite.Update)
	if runtime.GOOS == "windows" {
		suite.T().Skip("Skipping permission test on Windows, as CI on Windows is running as Administrator and is allowed to do EVERYTHING")
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// use unique exe
	ts.UseDistinctStateExes()

	// Todo This should not be necessary https://www.pivotaltracker.com/story/show/177865635
	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false, true)...))
	cp.ExpectExitCode(0)

	// Spoof modtime
	t := time.Now().Add(-25 * time.Hour)
	os.Chtimes(ts.ExecutablePath(), t, t)

	before := fileutils.ListDir(ts.Dirs.Config, false)

	cp = ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(suite.env(true, true)...), e2e.NonWriteableBinDir())
	cp.Expect("Updating State Tool to latest version available")
	cp.Expect(fmt.Sprintf("Version update to %s@", constants.BranchName))
	cp.ExpectExitCode(0)

	logs := suite.pollForUpdateInBackground(ts.Dirs.Config, before)
	suite.Assert().Contains(logs, "Installation failed")

	suite.versionCompare(ts, true, true, constants.Version, suite.Equal)
}

func TestUpdateIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode.")
	}
	suite.Run(t, new(UpdateIntegrationTestSuite))
}

func lockedProjectURL() string {
	return fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
}

func (suite *UpdateIntegrationTestSuite) addProjectFileWithWaitingScript(workDir string) {
	pjfile := projectfile.Project{
		Project: fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL),
		Scripts: []projectfile.Script{
			{Name: "wait", Value: "read -p \"Waiting for input\" -t $1", Conditional: "ne .OS.Name \"Windows\"", Language: "bash"},
			{Name: "wait", Value: "echo \"Waiting for input\"\ntimeout %1", Conditional: "eq .OS.Name \"Windows\"", Language: "cmd"},
		},
	}
	pjfile.SetPath(filepath.Join(workDir, constants.ConfigFileName))
	err := pjfile.Save(suite.cfg)
	suite.Require().NoError(err)
}
