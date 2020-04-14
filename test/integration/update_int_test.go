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
	cp := ts.SpawnWithOpts(e2e.WithArgs("--version"), e2e.AppendEnv(suite.env(disableUpdates)...))
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

	suite.versionCompare(ts, true, constants.VersionNumber, suite.Equal)
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdate() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.versionCompare(ts, false, constants.VersionNumber, suite.NotEqual)
}

func (suite *UpdateIntegrationTestSuite) TestLocked() {
	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
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

	suite.versionCompare(ts, false, constants.VersionNumber, suite.Equal)
}

func (suite *UpdateIntegrationTestSuite) TestUpdate() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(suite.env(true)...))
	// on master branch, we might already have the latest version available
	if os.Getenv("GIT_BRANCH") == "master" {
		cp.ExpectRe("(Update completed|You are using the latest version available)", 60*time.Second)
	} else {
		cp.Expect("Update completed", 60*time.Second)
	}
	cp.ExpectExitCode(0)

	suite.versionCompare(ts, false, constants.VersionNumber, suite.NotEqual)
}

func TestUpdateIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode.")
	}
	suite.Run(t, new(UpdateIntegrationTestSuite))
}
