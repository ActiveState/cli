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

var envs e2e.SpawnOptions

func (suite *UpdateIntegrationTestSuite) SetupTest() {
	envs = e2e.AppendEnv(
		"ACTIVESTATE_CLI_AUTO_UPDATE_TIMEOUT=10",
		"ACTIVESTATE_CLI_UPDATE_BRANCH=master")
}

func (suite *UpdateIntegrationTestSuite) getVersion(ts *e2e.Session, extraEnv ...string) string {
	cp := ts.SpawnWithOpts(e2e.WithArgs("--version"), envs, e2e.AppendEnv(extraEnv...))
	cp.Expect("ActiveState CLI version ")
	cp.Expect("Revision")
	cp.ExpectExitCode(0)
	regex := regexp.MustCompile(`\d+\.\d+\.\d+-(SHA)?[a-f0-9]+`)
	return regex.FindString(cp.TrimmedSnapshot())
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdateDisabled() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	extraEnv := "ACTIVESTATE_CLI_DISABLE_UPDATES=true"
	suite.NotEqual(constants.VersionNumber, suite.getVersion(ts, extraEnv), "Versions should match as auto-update should not have occurred")
}

func (suite *UpdateIntegrationTestSuite) TestAutoUpdate() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	extraEnv := "ACTIVESTATE_CLI_DISABLE_UPDATES=false"
	suite.NotEqual(constants.VersionNumber, suite.getVersion(ts, extraEnv), "Versions shouldn't match as auto-update should have occurred")
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

	extraEnv := "ACTIVESTATE_CLI_DISABLE_UPDATES=false"
	cp := ts.SpawnWithOpts(
		e2e.WithArgs("update", "--lock"),
		e2e.AppendEnv(extraEnv),
	)

	cp.Expect("Version locked at")
	cp.ExpectExitCode(0)

	suite.NotEqual(constants.VersionNumber, suite.getVersion(ts, extraEnv), "Versions should match because locking is enabled")
}

func (suite *UpdateIntegrationTestSuite) TestUpdate() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	extraEnv := "ACTIVESTATE_CLI_DISABLE_UPDATES=true"
	cp := ts.SpawnWithOpts(e2e.WithArgs("update"), e2e.AppendEnv(extraEnv))
	// on master branch, we might already have the latest version available
	if os.Getenv("GIT_BRANCH") == "master" {
		cp.ExpectRe("(Update completed|You are using the latest version available)", 60*time.Second)
	} else {
		cp.Expect("Update completed", 60*time.Second)
	}
	cp.ExpectExitCode(0)
	fmt.Println("Version after update: ", suite.getVersion(ts))

	suite.NotEqual(constants.VersionNumber, suite.getVersion(ts), "Versions shouldn't match as we ran update")
}

func TestUpdateIntegrationTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode.")
	}
	suite.Run(t, new(UpdateIntegrationTestSuite))
}
