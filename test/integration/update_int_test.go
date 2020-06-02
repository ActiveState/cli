package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type UpdateIntegrationTestSuite struct {
	suite.Suite
}

type matcherFunc func(expected interface{}, actual interface{}, msgAndArgs ...interface{}) bool

func (suite *UpdateIntegrationTestSuite) env(disableUpdates bool) []string {
	env := []string{
		"ACTIVESTATE_CLI_AUTO_UPDATE_TIMEOUT=10",
		"ACTIVESTATE_CLI_UPDATE_BRANCH=master",
	}

	if disableUpdates {
		env = append(env, "ACTIVESTATE_CLI_DISABLE_UPDATES=true")
	} else {
		env = append(env, "ACTIVESTATE_CLI_DISABLE_UPDATES=false")
	}

	return env
}

func (suite *UpdateIntegrationTestSuite) versionCompare(ts *e2e.Session, disableUpdates bool, expected string, matcher matcherFunc) {
	cp := ts.SpawnWithOpts(e2e.WithArgs("--version"), e2e.AppendEnv(suite.env(disableUpdates)...), e2e.ReUseExecutable())
	cp.Expect("ActiveState CLI version ")
	cp.Expect("Revision")
	cp.ExpectExitCode(0)
	regex := regexp.MustCompile(`\d+\.\d+\.\d+-(SHA)?[a-f0-9]+`)
	resultVersions := regex.FindAllString(cp.TrimmedSnapshot(), -1)

	suite.GreaterOrEqual(len(resultVersions), 1,
		fmt.Sprintf("Must have more than 0 matches (the first one being the 'Updating from X to Y' message, matched versions: %v, output:\n\n%s", resultVersions, cp.Snapshot()),
	)

	_ = matcher(expected, resultVersions[len(resultVersions)-1], fmt.Sprintf("Version could not be matched, output:\n\n%s", cp.Snapshot()))
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdateDisabled() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.versionCompare(ts, true, constants.Version, suite.Equal)
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdate() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.versionCompare(ts, false, constants.Version, suite.NotEqual)
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdateNoPermissions() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version"), e2e.AppendEnv(suite.env(false)...), e2e.NonWriteableBinDir())
	cp.Expect("Could not update to the latest available version of the state tool due to insufficient permissions")
	cp.Expect("ActiveState CLI version ")
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
	pjfile := projectfile.Project{
		Project: lockedProjectURL(),
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	pjfile.Save()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("update", "--lock"),
		e2e.AppendEnv(suite.env(false)...),
	)

	cp.Expect("Version locked at")
	cp.ExpectExitCode(0)

	suite.versionCompare(ts, false, constants.Version, suite.Equal)
}

func (suite *UpdateIntegrationTestSuite) TestUpdateLockedConfirmationNegative() {
	pjfile := projectfile.Project{
		Project: lockedProjectURL(),
		Version: constants.Version,
		Branch:  constants.BranchName,
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	pjfile.Save()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("update"),
		e2e.AppendEnv(suite.env(true)...),
	)
	cp.Expect("sure you want")
	cp.SendLine("n")
	cp.Expect("not confirm")
	cp.ExpectNotExitCode(0)
}

// TestUpdateLockedConfirmationPositive does not verify the effects of the
// update behavior. That is left to TestUpdate.
func (suite *UpdateIntegrationTestSuite) TestUpdateLockedConfirmationPositive() {
	pjfile := projectfile.Project{
		Project: lockedProjectURL(),
		Version: constants.Version,
		Branch:  constants.BranchName,
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	pjfile.Save()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("update"),
		e2e.AppendEnv(suite.env(true)...),
	)
	cp.Expect("sure you want")
	cp.SendLine("y")
	cp.Expect("Locked version updated")
	cp.ExpectExitCode(0)
}

// TestUpdateLockedConfirmationForce does not verify the effects of the
// update behavior. That is left to TestUpdate.
func (suite *UpdateIntegrationTestSuite) TestUpdateLockedConfirmationForce() {
	pjfile := projectfile.Project{
		Project: lockedProjectURL(),
		Version: constants.Version,
		Branch:  constants.BranchName,
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	pjfile.SetPath(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	pjfile.Save()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("update", "--force"),
		e2e.AppendEnv(suite.env(true)...),
	)
	cp.Expect("Locked version updated")
	cp.ExpectExitCode(0)
}

func (suite *UpdateIntegrationTestSuite) TestUpdate() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(suite.env(false)...))
	// on master branch, we might already have the latest version available
	if os.Getenv("GIT_BRANCH") == "master" {
		cp.ExpectRe("(Version updated|You are using the latest version available)", 60*time.Second)
	} else {
		cp.Expect("Downloading latest version of the state tool")
		cp.Expect("Version updated", 60*time.Second)

	}
	cp.ExpectExitCode(0)

	if os.Getenv("GIT_BRANCH") != "master" {
		regex := regexp.MustCompile(`\d+\.\d+\.\d+-(SHA)?[a-f0-9]+`)
		resultVersions := regex.FindAllString(cp.TrimmedSnapshot(), -1)

		suite.GreaterOrEqual(len(resultVersions), 1,
			fmt.Sprintf("Must have more than 0 matches (the first one being the 'Updating from X to Y' message, matched versions: %v, output:\n\n%s", resultVersions, cp.Snapshot()),
		)

		suite.NotEqual(constants.Version, resultVersions[len(resultVersions)-1], fmt.Sprintf("Expected to update to a new a new version:\n\n%s", cp.Snapshot()))
	}

	suite.versionCompare(ts, true, constants.Version, suite.NotEqual)
}

func (suite *UpdateIntegrationTestSuite) TestUpdateNoPermissions() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(suite.env(true)...), e2e.NonWriteableBinDir())
	// on master branch, we might already have the latest version available
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
