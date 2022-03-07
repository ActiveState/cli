package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ForkIntegrationTestSuite struct {
	tagsuite.Suite
	username string
}

func (suite *ForkIntegrationTestSuite) cleanup(ts *e2e.Session) {
	cp := ts.Spawn(tagsuite.Auth, "logout")
	cp.ExpectExitCode(0)
	ts.Close()
}

func (suite *ForkIntegrationTestSuite) TestFork() {
	suite.OnlyRunForTags(tagsuite.Fork)
	ts := e2e.New(suite.T(), false)
	defer suite.cleanup(ts)

	username := ts.CreateNewUser()

	cp := ts.Spawn("fork", "ActiveState-CLI/Python3", "--name", "Test-Python3", "--org", username)
	cp.Expect("fork has been successfully created")
	cp.ExpectExitCode(0)

	// Check if we error out on conflicts properly
	cp = ts.Spawn("fork", "ActiveState-CLI/Python3", "--name", "Test-Python3", "--org", username)
	cp.Expect(`Could not create project`)
	cp.ExpectExitCode(1)
}

func (suite *ForkIntegrationTestSuite) TestFork_FailNameExists() {
	suite.OnlyRunForTags(tagsuite.Fork)
	ts := e2e.New(suite.T(), false)
	defer suite.cleanup(ts)
	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("fork", "ActiveState-CLI/Python3", "--org", e2e.PersistentUsername),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Could not create project:", 30*time.Second)
	cp.Expect("You already have a project with the name 'Python3'.", 30*time.Second)
	cp.ExpectNotExitCode(0)
	suite.NotContains(cp.TrimmedSnapshot(), "Successfully forked project")
}

func TestForkIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ForkIntegrationTestSuite))
}
