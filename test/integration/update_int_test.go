package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/appinfo"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/suite"
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

	ts.UseDistinctStateExes()

	suite.testUpdate(ts, ts.Dirs.Bin)
}

func (suite *UpdateIntegrationTestSuite) testUpdate(ts *e2e.Session, baseDir string, opts... e2e.SpawnOptions) {
	cfg, err := config.NewCustom(ts.Dirs.Config, singlethread.New(), true)
	suite.Require().NoError(err)
	defer cfg.Close()

	stateExe := appinfo.StateApp(baseDir)

	spawnOpts := []e2e.SpawnOptions{
		e2e.WithArgs("update"),
		e2e.AppendEnv(suite.env(false, true)...),
	}
	if opts != nil {
		spawnOpts = append(spawnOpts, opts...)
	}

	cp := ts.SpawnCmdWithOpts(stateExe.Exec(), spawnOpts...)
	cp.Expect("Updating State Tool to latest version available")
	cp.Expect("Installing Update")
	cp.Expect("Done")
	cp.ExpectExitCode(0)
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

			updateArgs := []string{"update", "--set-channel", tt.Channel}
			if tt.Version != "" {
				updateArgs = append(updateArgs, "--set-version", tt.Version)
			}
			cp := ts.SpawnWithOpts(
				e2e.WithArgs(updateArgs...),
				e2e.AppendEnv(suite.env(false, false)...),
			)
			cp.Expect("Updating")
			cp.ExpectExitCode(0)

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

			// ..
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

	ts.UseDistinctStateExes()

	suite.testAutoUpdate(ts, ts.Dirs.Bin)
}

func (suite *UpdateIntegrationTestSuite) testAutoUpdate(ts *e2e.Session, baseDir string, opts... e2e.SpawnOptions) {
	stateExe := appinfo.StateApp(baseDir)

	fakeHome := filepath.Join(ts.Dirs.Work, "home")
	suite.Require().NoError(fileutils.Mkdir(fakeHome))

	// Spoof modtime
	t := time.Now().Add(-25 * time.Hour)
	os.Chtimes(ts.ExecutablePath(), t, t)

	spawnOpts := []e2e.SpawnOptions{
		e2e.WithArgs("--version"),
		e2e.AppendEnv(suite.env(false, true)...),
		e2e.AppendEnv(fmt.Sprintf("HOME=%s", fakeHome)),
		e2e.AppendEnv("VERBOSE=true"),
	}
	if opts != nil {
		spawnOpts = append(spawnOpts, opts...)
	}

	cp := ts.SpawnCmdWithOpts(stateExe.Exec(), spawnOpts...)
	cp.Expect("Auto Update")
	cp.Expect("Updating State Tool")
	cp.Expect("Update installed")
	cp.ExpectExitCode(0)
}



func (suite *UpdateIntegrationTestSuite) installLatestReleaseVersion(ts *e2e.Session, dir string) {
	var cp *termtest.ConsoleProcess
	if runtime.GOOS != "windows" {
		oneLiner := fmt.Sprintf("sh <(curl -q https://platform.activestate.com/dl/cli/pdli01/install.sh) -f -n -t %s", dir)
		cp = ts.SpawnCmdWithOpts(
			"bash", e2e.WithArgs("-c", oneLiner),
		)
	} else {
		oneLiner := `powershell -Command "& $([scriptblock]::Create((New-Object Net.WebClient).DownloadString('https://platform.activestate.com/dl/cli/pdli01/install.ps1')))"`
		oneLiner += fmt.Sprintf(" -f -n -t %s", dir)
		cp = ts.SpawnCmdWithOpts("cmd.exe", e2e.WithArgs("/c", oneLiner),
			e2e.AppendEnv("SHELL="),
		)
	}
	cp.Expect("Installation Complete", time.Second*30)
	cp.ExpectExitCode(0)

	suite.FileExists(appinfo.StateApp(dir).Exec())
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdateToCurrent() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	installDir := filepath.Join(ts.Dirs.Work, "install")
	fileutils.MkdirUnlessExists(installDir)

	suite.installLatestReleaseVersion(ts, installDir)

	suite.testAutoUpdate(ts, installDir, e2e.AppendEnv(fmt.Sprintf("ACTIVESTATE_CLI_UPDATE_BRANCH=%s", constants.BranchName)))
}

func (suite *UpdateIntegrationTestSuite) TestUpdateToCurrent() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	installDir := filepath.Join(ts.Dirs.Work, "install")
	fileutils.MkdirUnlessExists(installDir)

	suite.installLatestReleaseVersion(ts, installDir)

	suite.testUpdate(ts, installDir, e2e.AppendEnv(fmt.Sprintf("ACTIVESTATE_CLI_UPDATE_BRANCH=%s", constants.BranchName)))
}