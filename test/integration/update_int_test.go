package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/internal/updater"
)

type UpdateIntegrationTestSuite struct {
	tagsuite.Suite
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

// env prepares environment variables for the test
// disableUpdates prevents all update code from running
// testUpdate directs to the locally running update directory and requires that a test update bundles has been generated with `state run generate-test-update`
func (suite *UpdateIntegrationTestSuite) env(disableUpdates, forceUpdate bool) []string {
	env := []string{}

	if disableUpdates {
		env = append(env, "ACTIVESTATE_CLI_DISABLE_UPDATES=true")
	} else {
		env = append(env, "ACTIVESTATE_CLI_DISABLE_UPDATES=false")
	}

	if forceUpdate {
		env = append(env, "ACTIVESTATE_FORCE_UPDATE=true")
	}

	dir, err := ioutil.TempDir("", "system*")
	suite.NoError(err)
	env = append(env, fmt.Sprintf("%s=%s", constants.OverwriteDefaultSystemPathEnvVarName, dir))

	return env
}

func (suite *UpdateIntegrationTestSuite) versionCompare(ts *e2e.Session, expected string, matcher matcherFunc) {
	type versionData struct {
		Version string `json:"version"`
	}

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExes()

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"), e2e.AppendEnv(suite.env(true, false)...))
	cp.ExpectExitCode(0)

	version := versionData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &version)

	matcher(expected, version.Version, fmt.Sprintf("Version could not be matched, output:\n\n%s", out))
}

func (suite *UpdateIntegrationTestSuite) branchCompare(ts *e2e.Session, expected string, matcher matcherFunc) {
	type branchData struct {
		Branch string `json:"branch"`
	}

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExesLegacy()

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"), e2e.AppendEnv(suite.env(true, false)...))
	cp.ExpectExitCode(0, 30*time.Second)

	branch := branchData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &branch)

	matcher(expected, branch.Branch, fmt.Sprintf("Branch could not be matched, output:\n\n%s", out))
}

func (suite *UpdateIntegrationTestSuite) TestUpdateAvailable() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), false)
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


func (suite *UpdateIntegrationTestSuite) TestUpdate() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	defer cfg.Close()

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExes()

	// Todo This should not be necessary https://www.pivotaltracker.com/story/show/177865635
	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false, true)...))
	cp.ExpectExitCode(0)

	fakeHome := filepath.Join(ts.Dirs.Work, "home")
	err = fileutils.Mkdir(fakeHome)
	suite.Require().NoError(err)

	before := fileutils.ListDirSimple(ts.Dirs.Config, false)

	cp = ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(suite.env(false, true)...), e2e.AppendEnv(fmt.Sprintf("HOME=%s", fakeHome)))
	cp.Expect("Updating State Tool to latest version available")
	cp.Expect(fmt.Sprintf("Version update to %s@", constants.BranchName))
	cp.ExpectExitCode(0)

	logs := suite.pollForUpdateInBackground(ts.Dirs.Config, before)
	suite.Assert().Contains(logs, "was successful")
}

func (suite *UpdateIntegrationTestSuite) TestUpdateChannel() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	tests := []struct {
		Name       string
		Channel    string
		Version    string
	}{
		{"release-channel", "release", ""},
		{"specific-update", targetBranch, specificVersion},
	}

	for _, tt := range tests {
		suite.Run(tt.Name, func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()

			// Ensure we always use a unique exe for updates
			ts.UseDistinctStateExes()

			// Todo This should not be necessary https://www.pivotaltracker.com/story/show/177865635
			cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(true, false)...))
			cp.ExpectExitCode(0)

			before := fileutils.ListDirSimple(ts.Dirs.Config, false)

			info, err := os.Stat(ts.Exe)
			suite.Require().NoError(err)
			modTime := info.ModTime()

			updateArgs := []string{"update", "--set-channel", tt.Channel}
			if tt.Version != "" {
				updateArgs = append(updateArgs, "--set-version", tt.Version)
			}
			cp = ts.SpawnWithOpts(
				e2e.WithArgs(updateArgs...),
				e2e.AppendEnv(suite.env(false, false)...),
			)
			if tt.Version == "" {
				cp.Expect("Updating State Tool to latest version available")
			} else {
				cp.Expect("Updating State Tool to version")
			}
			cp.Expect(fmt.Sprintf("Version update to %s@", tt.Channel))
			cp.ExpectExitCode(0)

			logs := suite.pollForUpdateInBackground(ts.Dirs.Config, before)
			suite.Assert().Contains(logs, "was successful")

			// Check for state tool executable to be updated
			updated := false
			// wait for up to two minutes for the State Tool to get modified
			for x := 0; x < 600; x++ {
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
			suite.Require().True(updated, "Timeout: Expected the State Tool to get modified. Output: %s", cp.Snapshot())

			suite.branchCompare(ts, tt.Channel, suite.Equal)

			if tt.Version != "" {
				suite.versionCompare(ts, tt.Version, suite.Equal)
			}
		})
	}
}


func (suite *UpdateIntegrationTestSuite) TestUpdateTags() {
	// Disabled, waiting for - https://www.pivotaltracker.com/story/show/179646813
	suite.T().Skip("Disabled for now")
	suite.OnlyRunForTags(tagsuite.Update)

	tagName := "experiment"

	tests := []struct {
		name          string
		tagged        bool
		expectSuccess bool
	}{
		{"update-to-tag", false, true},
		{"update-with-tag", true, false},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			ts := e2e.New(suite.T(), false)
			defer ts.Close()
			// use unique exe
			ts.UseDistinctStateExes()

			fakeHome := filepath.Join(ts.Dirs.Work, "home")
			err := fileutils.Mkdir(fakeHome)
			suite.Require().NoError(err)

			cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
			suite.Require().NoError(err)
			defer cfg.Close()

			if tt.tagged {
				err := cfg.Set(updater.CfgUpdateTag, tagName)
				suite.Require().NoError(err)
				suite.Assert().Equal(tagName, cfg.GetString(updater.CfgUpdateTag))
			}

			before := fileutils.ListDirSimple(ts.Dirs.Config, false)
			cp := ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(suite.env(true, true)...), e2e.AppendEnv(fmt.Sprintf("HOME=%s", fakeHome)))
			cp.Expect("Updating State Tool to latest version available")
			if tt.expectSuccess {
				cp.Expect(fmt.Sprintf("Version update to %s@", constants.BranchName))
				cp.ExpectExitCode(0)
				logs := suite.pollForUpdateInBackground(ts.Dirs.Config, before)
				suite.Assert().Contains(logs, "was successful")
				suite.Assert().Equal("experiment", cfg.GetString(updater.CfgUpdateTag))
				suite.versionCompare(ts, constants.Version, suite.NotEqual)
			} else {
				cp.ExpectLongString("404 Not Found")
				cp.ExpectExitCode(1)
			}
		})
	}
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

func (suite *UpdateIntegrationTestSuite) TestAutoUpdate() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	// use unique exe
	ts.UseDistinctStateExes()

	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false, true)...))
	cp.ExpectExitCode(0)

	before := fileutils.ListDirSimple(ts.Dirs.Config, false)

	fakeHome := filepath.Join(ts.Dirs.Work, "home")
	suite.Require().NoError(fileutils.Mkdir(fakeHome))

	// Spoof modtime
	t := time.Now().Add(-25 * time.Hour)
	os.Chtimes(ts.ExecutablePath(), t, t)

	cp = ts.SpawnWithOpts(e2e.WithArgs("--version"), e2e.AppendEnv(suite.env(false, true)...),
		e2e.AppendEnv(fmt.Sprintf("HOME=%s", fakeHome)),
		e2e.AppendEnv("VERBOSE=true"))
	cp.Expect("Auto Update")
	cp.Expect("Updating State Tool")
	cp.ExpectExitCode(0)

	suite.pollForUpdateInBackground(ts.Dirs.Config, before)
}

func (suite *UpdateIntegrationTestSuite) pollForUpdateInBackground(configDir string, beforeFiles []string) string {
	for i := 0; i < 10; i++ {
		after := fileutils.ListDirSimple(configDir, false)
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