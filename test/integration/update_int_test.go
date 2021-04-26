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
	"regexp"
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

	suite.cfg, err = config.Get()
	suite.Require().NoError(err)
}

func (suite *UpdateIntegrationTestSuite) AfterTest(suiteName, testName string) {
	err := suite.server.Shutdown(context.Background())
	suite.Require().NoError(err)
}

// env prepares environment variables for the test
// disableUpdates prevents all update code from running
// testUpdate directs to the locally running update directory and requires that a test update bundles has been generated with `state run generate-test-update`
func (suite *UpdateIntegrationTestSuite) env(disableUpdates, testUpdate bool) []string {
	env := []string{
		"ACTIVESTATE_CLI_AUTO_UPDATE_TIMEOUT=10",
	}

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
	ts.UseDistinctStateExes()

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"), e2e.AppendEnv(suite.env(disableUpdates, testUpdate)...))
	cp.ExpectExitCode(0, 30*time.Second)

	branch := branchData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &branch)

	matcher(expected, branch.Branch, fmt.Sprintf("Branch could not be matched, output:\n\n%s", out))
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdateDisabled() {
	suite.OnlyRunForTags(tagsuite.Update)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.versionCompare(ts, true, true, constants.Version, suite.Equal)
}

func (suite *UpdateIntegrationTestSuite) TestNoAutoUpdate() {
	suite.OnlyRunForTags(tagsuite.Update)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// update should not run because the exe is less than a day old
	suite.versionCompare(ts, false, true, constants.Version, suite.Equal)
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdate() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	// use unique exe
	ts.UseDistinctStateExes()

	// Todo This should not be necessary https://www.pivotaltracker.com/story/show/177865635
	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false, true)...))
	cp.ExpectExitCode(0)

	// Spoof modtime
	t := time.Now().Add(-25 * time.Hour)
	os.Chtimes(ts.ExecutablePath(), t, t)

	// update should run because the exe is more than a day old
	suite.versionCompare(ts, false, true, constants.Version, suite.NotEqual)
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdateNoPermissions() {
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

	cp = ts.SpawnWithOpts(e2e.WithArgs("--version"), e2e.AppendEnv(suite.env(false, true)...), e2e.NonWriteableBinDir())
	cp.Expect("ActiveState CLI")
	cp.Expect("Revision")
	cp.ExpectExitCode(0)
	regex := regexp.MustCompile(`\d+\.\d+\.\d+-(SHA)?[a-f0-9]+`)
	resultVersions := regex.FindAllString(cp.TrimmedSnapshot(), -1)

	suite.GreaterOrEqual(len(resultVersions), 1,
		fmt.Sprintf("Must have more than 0 matches (the first one being the 'Updating from X to Y' message, matched versions: %v, output:\n\n%s", resultVersions, cp.Snapshot()),
	)

	suite.Equal(constants.Version, resultVersions[len(resultVersions)-1], "Did not expect updated version, output:\n\n%s", cp.Snapshot())
}

func (suite *UpdateIntegrationTestSuite) pollForUpdateInBackground(output string) string {
	regex := regexp.MustCompile(`Refer to log *file *(.*) *for *progress.`)
	resultLogfile := regex.FindStringSubmatch(output)

	suite.Require().Equal(len(resultLogfile), 2, "expected to have logfile in output %s", output)

	return suite.pollForUpdateFromLogfile(strings.TrimSpace(resultLogfile[1]))
}

func (suite *UpdateIntegrationTestSuite) pollForUpdateFromLogfile(logFile string) string {
	// poll for successful auto-update
	for i := 0; i < 30; i++ {
		time.Sleep(time.Millisecond * 200)

		logs, err := ioutil.ReadFile(logFile)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		suite.NoError(err)
		if strings.Contains(string(logs), "was successful") || strings.Contains(string(logs), "Installation failed") {
			return string(logs)
		}
	}

	suite.T().Errorf("did not find logFile %s", logFile)
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

			cp = ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(suite.env(false, tt.TestUpdate)...), e2e.AppendEnv(fmt.Sprintf("HOME=%s", fakeHome)))
			cp.Expect("Updating State Tool to latest version available")
			cp.Expect(fmt.Sprintf("Version update to %s@", constants.BranchName))
			cp.ExpectExitCode(0)

			logs := suite.pollForUpdateInBackground(cp.TrimmedSnapshot())

			// tell background process to stop...
			close(stopBg)
			// ...and wait for it
			wg.Wait()

			suite.Assert().Contains(logs, "was successful")
			suite.versionCompare(ts, true, tt.TestUpdate, constants.Version, suite.NotEqual)

			if tt.ExpectBackupCleaned {
				suite.Assert().Contains(logs, "Removed all backup files.")
			} else {
				suite.Assert().Contains(logs, "Failed to remove backup files")
			}
		})
	}
}

func (suite *UpdateIntegrationTestSuite) TestUpdateChannel() {
	tests := []struct {
		Name       string
		TestUpdate bool
		Channel    string
	}{
		{"test-update", true, testBranch},
		{"release-channel", false, targetBranch},
	}
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	for _, tt := range tests {
		if !tt.TestUpdate {
			suite.T().Skipf("This requires a new update bundle to be deployed to the %s channel", targetBranch)
		}
		suite.Run(tt.Name, func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			// Ensure we always use a unique exe for updates
			ts.UseDistinctStateExes()

			// Todo This should not be necessary https://www.pivotaltracker.com/story/show/177865635
			cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false, tt.TestUpdate)...))
			cp.ExpectExitCode(0)

			cp = ts.SpawnWithOpts(e2e.WithArgs("update", "--set-channel", tt.Channel), e2e.AppendEnv(suite.env(false, tt.TestUpdate)...))
			cp.Expect("Updating State Tool to latest version available")
			cp.Expect(fmt.Sprintf("Version update to %s@", tt.Channel))
			cp.ExpectExitCode(0)

			logs := suite.pollForUpdateInBackground(cp.TrimmedSnapshot())
			suite.Assert().Contains(logs, "was successful")

			suite.branchCompare(ts, false, tt.TestUpdate, tt.Channel, suite.Equal)
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

	cp = ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(suite.env(true, true)...), e2e.NonWriteableBinDir())
	cp.Expect("Updating State Tool to latest version available")
	cp.Expect(fmt.Sprintf("Version update to %s@", constants.BranchName))
	cp.ExpectExitCode(0)

	suite.pollForUpdateInBackground(cp.TrimmedSnapshot())
	logs := suite.pollForUpdateInBackground(cp.TrimmedSnapshot())
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
