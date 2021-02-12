package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type UpdateIntegrationTestSuite struct {
	tagsuite.Suite
	cfg projectfile.ConfigGetter
}

type matcherFunc func(expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool

var targetBranch = "release"

func init() {
	if constants.BranchName == targetBranch {
		targetBranch = "master"
	}
}

func (suite *UpdateIntegrationTestSuite) BeforeTest(suiteName, testName string) {
	var err error
	suite.cfg, err = config.Get()
	suite.Require().NoError(err)
}

func (suite *UpdateIntegrationTestSuite) env(disableUpdates bool) []string {
	env := []string{
		"ACTIVESTATE_CLI_AUTO_UPDATE_TIMEOUT=10",
		"ACTIVESTATE_CLI_UPDATE_BRANCH=" + targetBranch,
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
	ts.UseDistinctStateExe()

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version", "--output=json"), e2e.AppendEnv(suite.env(disableUpdates)...))
	cp.ExpectExitCode(0)

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
	ts.UseDistinctStateExe()

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
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// use unique exe
	ts.UseDistinctStateExe()

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
	ts.UseDistinctStateExe()

	// Spoof modtime
	t := time.Now().Add(-25 * time.Hour)
	os.Chtimes(ts.ExecutablePath(), t, t)

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version"), e2e.AppendEnv(suite.env(false)...), e2e.NonWriteableBinDir())
	cp.Expect("insufficient permissions")
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

func (suite *UpdateIntegrationTestSuite) TestLocked() {
	suite.T().SkipNow() // https://www.pivotaltracker.com/story/show/176926586
	suite.OnlyRunForTags(tagsuite.Update)
	pjfile := projectfile.Project{
		Project: lockedProjectURL(),
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExe()

	pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	pjfile.Save(suite.cfg)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("update", "lock"),
		e2e.AppendEnv(suite.env(false)...),
	)

	cp.Expect("Version locked at")
	cp.ExpectExitCode(0)

	suite.versionCompare(ts, false, constants.Version, suite.NotEqual)
}

func (suite *UpdateIntegrationTestSuite) TestLockedChannel() {
	suite.T().SkipNow() // https://www.pivotaltracker.com/story/show/176926586
	suite.OnlyRunForTags(tagsuite.Update)
	pjfile := projectfile.Project{
		Project: lockedProjectURL(),
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExe()

	pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	pjfile.Save(suite.cfg)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("update", "lock", "--set-channel", targetBranch),
		e2e.AppendEnv(suite.env(false)...),
	)

	cp.Expect("Version locked at")
	cp.Expect(targetBranch + "@")
	cp.ExpectExitCode(0)

	suite.branchCompare(ts, false, targetBranch, suite.Equal)
}

func (suite *UpdateIntegrationTestSuite) TestLockedChannelVersion() {
	suite.OnlyRunForTags(tagsuite.Update)
	pjfile := projectfile.Project{
		Project: lockedProjectURL(),
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExe()

	yamlPath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)
	pjfile.SetPath(yamlPath)
	pjfile.Save(suite.cfg)

	lock := targetBranch + "@latest"
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("update", "lock", "--set-channel", lock),
		e2e.AppendEnv(suite.env(false)...),
	)

	cp.Expect("Version locked at")
	cp.Expect(targetBranch + "@")
	cp.ExpectExitCode(0)

	yamlContents, err := fileutils.ReadFile(yamlPath)
	suite.Require().NoError(err)
	suite.Contains(string(yamlContents), lock)
}

func (suite *UpdateIntegrationTestSuite) TestUpdateLockedConfirmationNegative() {
	suite.OnlyRunForTags(tagsuite.Update)
	pjfile := projectfile.Project{
		Project: lockedProjectURL(),
		Lock:    fmt.Sprintf("%s@%s", constants.BranchName, constants.Version),
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExe()

	pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	pjfile.Save(suite.cfg)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("update", "lock"),
		e2e.AppendEnv(suite.env(true)...),
	)
	cp.Expect("sure you want")
	cp.Send("n")
	cp.Expect("not confirm")
	cp.ExpectNotExitCode(0)
}

// TestUpdateLockedConfirmationPositive does not verify the effects of the
// update behavior. That is left to TestUpdate.
func (suite *UpdateIntegrationTestSuite) TestUpdateLockedConfirmationPositive() {
	suite.OnlyRunForTags(tagsuite.Update)
	pjfile := projectfile.Project{
		Project: lockedProjectURL(),
		Lock:    fmt.Sprintf("%s@%s", constants.BranchName, constants.Version),
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExe()

	pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	pjfile.Save(suite.cfg)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("update", "lock"),
		e2e.AppendEnv(suite.env(true)...),
	)
	cp.Expect("sure you want")
	cp.Send("y")
	cp.Expect("Version locked at")
	cp.ExpectExitCode(0)
}

// TestUpdateLockedConfirmationForce does not verify the effects of the
// update behavior. That is left to TestUpdate.
func (suite *UpdateIntegrationTestSuite) TestUpdateLockedConfirmationForce() {
	suite.OnlyRunForTags(tagsuite.Update)
	pjfile := projectfile.Project{
		Project: lockedProjectURL(),
		Lock:    fmt.Sprintf("%s@%s", constants.BranchName, constants.Version),
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExe()

	pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	pjfile.Save(suite.cfg)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("update", "lock", "--force"),
		e2e.AppendEnv(suite.env(true)...),
	)
	cp.Expect("Version locked at")
	cp.ExpectExitCode(0)
}

func (suite *UpdateIntegrationTestSuite) TestUpdate() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExe()

	cp := ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(suite.env(false)...))
	cp.Expect("Updating state tool:  Downloading latest version")
	cp.Expect("Version updated", 60*time.Second)
	cp.ExpectExitCode(0)

	regex := regexp.MustCompile(`\d+\.\d+\.\d+-(SHA)?[a-f0-9]+`)
	resultVersions := regex.FindAllString(cp.TrimmedSnapshot(), -1)

	suite.GreaterOrEqual(len(resultVersions), 1,
		fmt.Sprintf("Must have more than 0 matches (the first one being the 'Updating from X to Y' message, matched versions: %v, output:\n\n%s", resultVersions, cp.Snapshot()),
	)

	suite.NotEqual(constants.Version, resultVersions[len(resultVersions)-1], fmt.Sprintf("Expected to update to a new a new version:\n\n%s", cp.Snapshot()))

	suite.versionCompare(ts, true, constants.Version, suite.NotEqual)
}

func (suite *UpdateIntegrationTestSuite) TestUpdateChannel() {
	suite.OnlyRunForTags(tagsuite.Update, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Ensure we always use a unique exe for updates
	ts.UseDistinctStateExe()

	cp := ts.SpawnWithOpts(e2e.WithArgs("update", "--set-channel", targetBranch), e2e.AppendEnv(suite.env(false)...))
	cp.Expect("Updating state tool:  Downloading latest version")
	cp.Expect("Version updated", 60*time.Second)
	cp.Expect(targetBranch+"@", 60*time.Second)
	cp.ExpectExitCode(0)

	suite.branchCompare(ts, false, targetBranch, suite.Equal)
}

func (suite *UpdateIntegrationTestSuite) TestUpdateNoPermissions() {
	suite.OnlyRunForTags(tagsuite.Update)
	if runtime.GOOS == "windows" {
		suite.T().Skip("Skipping permission test on Windows, as CI on Windows is running as Administrator and is allowed to do EVERYTHING")
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// use unique exe
	ts.UseDistinctStateExe()

	// Spoof modtime
	t := time.Now().Add(-25 * time.Hour)
	os.Chtimes(ts.ExecutablePath(), t, t)

	cp := ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(suite.env(true)...), e2e.NonWriteableBinDir())
	cp.Expect("Update failed due to permission error")
	cp.ExpectNotExitCode(0)

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
