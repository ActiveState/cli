package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
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

var targetBranch = "release"
var testBranch = "test-channel"

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

func (suite *UpdateIntegrationTestSuite) env(disableUpdates bool) []string {
	env := []string{
		fmt.Sprintf("_TEST_UPDATE_URL=http://localhost:%s/", testPort),
		"ACTIVESTATE_CLI_AUTO_UPDATE_TIMEOUT=10",
	}

	if disableUpdates {
		env = append(env, "ACTIVESTATE_CLI_DISABLE_UPDATES=true")
	} else {
		env = append(env, "ACTIVESTATE_CLI_DISABLE_UPDATES=false")
	}

	return env
}

func (suite *UpdateIntegrationTestSuite) versionCompare(ts *e2e.Session, disableUpdates bool, expected string, matcher matcherFunc) {
	type versionData struct {
		Version string `json:"version"`
	}

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExes()

	before := fileutils.ListDir(ts.Dirs.Config, false)

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"), e2e.AppendEnv(suite.env(disableUpdates)...))
	cp.ExpectExitCode(0)

	// short timeout to wait for installation log file to be created
	time.Sleep(500 * time.Millisecond)
	after := fileutils.ListDir(ts.Dirs.Config, false)
	onlyAfter, _ := funk.Difference(after, before)
	logFile, ok := funk.FindString(onlyAfter.([]string), func(s string) bool { return strings.HasPrefix(filepath.Base(s), "state-installer") })
	if ok {
		suite.pollForUpdateFromLogfile(logFile)
		cp = ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"), e2e.AppendEnv(suite.env(disableUpdates)...))
		cp.ExpectExitCode(0)
	}

	version := versionData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &version)

	matcher(expected, version.Version, fmt.Sprintf("Version could not be matched, output:\n\n%s", out))
}

func (suite *UpdateIntegrationTestSuite) branchCompare(ts *e2e.Session, disableUpdates bool, expected string, matcher matcherFunc) {
	type branchData struct {
		Branch string `json:"branch"`
	}

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExes()

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"), e2e.AppendEnv(suite.env(disableUpdates)...))
	cp.ExpectExitCode(0)

	branch := branchData{}
	out := strings.Trim(cp.TrimmedSnapshot(), "\x00")
	json.Unmarshal([]byte(out), &branch)

	matcher(expected, branch.Branch, fmt.Sprintf("Branch could not be matched, output:\n\n%s", out))
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdateDisabled() {
	suite.OnlyRunForTags(tagsuite.Update)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.versionCompare(ts, true, constants.Version, suite.Equal)
}

func (suite *UpdateIntegrationTestSuite) TestNoAutoUpdate() {
	suite.OnlyRunForTags(tagsuite.Update)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// update should not run because the exe is less than a day old
	suite.versionCompare(ts, false, constants.Version, suite.Equal)
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdate() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)
	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	// use unique exe
	ts.UseDistinctStateExes()

	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false)...))
	cp.ExpectExitCode(0)

	// Spoof modtime
	t := time.Now().Add(-25 * time.Hour)
	os.Chtimes(ts.ExecutablePath(), t, t)

	// update should run because the exe is more than a day old
	suite.versionCompare(ts, false, constants.Version, suite.NotEqual)
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

	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false)...))
	cp.ExpectExitCode(0)

	// Spoof modtime
	t := time.Now().Add(-25 * time.Hour)
	os.Chtimes(ts.ExecutablePath(), t, t)

	cp = ts.SpawnWithOpts(e2e.WithArgs("--version"), e2e.AppendEnv(suite.env(false)...), e2e.NonWriteableBinDir())
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

func (suite *UpdateIntegrationTestSuite) TestUpdateLock() {
	suite.OnlyRunForTags(tagsuite.Update)
	pjfile := projectfile.Project{
		Project: lockedProjectURL(),
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExes()

	pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	pjfile.Save(suite.cfg)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("update", "lock"),
		e2e.AppendEnv(suite.env(false)...),
	)

	cp.ExpectLongString("This version of the State Tool does not support version locking")
	cp.ExpectExitCode(1)

	suite.versionCompare(ts, false, constants.Version, suite.Equal)
}

func (suite *UpdateIntegrationTestSuite) pollForUpdateInBackground(output string) string {
	regex := regexp.MustCompile(`Refer to logfile (.*) for progress.`)
	resultLogfile := regex.FindStringSubmatch(output)

	suite.Equal(len(resultLogfile), 2, "expected to have logfile in output.")

	return suite.pollForUpdateFromLogfile(resultLogfile[1])
}

func (suite *UpdateIntegrationTestSuite) pollForUpdateFromLogfile(logFile string) string {
	// poll for successful auto-update
	for i := 0; i < 20; i++ {
		time.Sleep(time.Millisecond * 200)

		logs, err := ioutil.ReadFile(logFile)
		suite.NoError(err)
		if strings.Contains(string(logs), "was successful") || strings.Contains(string(logs), "Installation failed") {
			return string(logs)
		}
	}

	return ""
}

func (suite *UpdateIntegrationTestSuite) TestUpdate() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExes()

	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false)...))
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(suite.env(false)...))
	cp.Expect("Updating State Tool to latest version available")
	cp.Expect(fmt.Sprintf("Version updating to %s@99.99.9999 in the background", constants.BranchName))
	cp.ExpectExitCode(0)

	logs := suite.pollForUpdateInBackground(cp.TrimmedSnapshot())
	suite.Assert().Contains(logs, "was successful")

	suite.versionCompare(ts, true, constants.Version, suite.NotEqual)
}

func (suite *UpdateIntegrationTestSuite) TestUpdateChannel() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExes()

	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false)...))
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(e2e.WithArgs("update", "--set-channel", testBranch), e2e.AppendEnv(suite.env(false)...))
	cp.Expect("Updating State Tool to latest version available")
	cp.Expect(fmt.Sprintf("Version updating to %s@99.99.9999 in the background", testBranch))
	cp.ExpectExitCode(0)

	logs := suite.pollForUpdateInBackground(cp.TrimmedSnapshot())
	suite.Assert().Contains(logs, "was successful")

	suite.branchCompare(ts, false, testBranch, suite.Equal)
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

	cp := ts.SpawnCmdWithOpts(ts.SvcExe, e2e.WithArgs("start"), e2e.AppendEnv(suite.env(false)...))
	cp.ExpectExitCode(0)

	// Spoof modtime
	t := time.Now().Add(-25 * time.Hour)
	os.Chtimes(ts.ExecutablePath(), t, t)

	cp = ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(suite.env(true)...), e2e.NonWriteableBinDir())
	cp.Expect("Updating State Tool to latest version available")
	cp.Expect(fmt.Sprintf("Version updating to %s@99.99.9999 in the background", constants.BranchName))
	cp.ExpectExitCode(0)

	suite.pollForUpdateInBackground(cp.TrimmedSnapshot())
	logs := suite.pollForUpdateInBackground(cp.TrimmedSnapshot())
	suite.Assert().Contains(logs, "Installation failed")

	suite.versionCompare(ts, true, constants.Version, suite.Equal)
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
