package integration

import (
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type ForkIntegrationTestSuite struct {
	suite.Suite
	username string
}

func (suite *ForkIntegrationTestSuite) cleanup(ts *e2e.Session) {
	cp := ts.Spawn("auth", "logout")
	cp.ExpectExitCode(0)
	ts.Close()
}

func (suite *ForkIntegrationTestSuite) TestFork_FailNameExists() {
	ts := e2e.New(suite.T(), false)
	defer suite.cleanup(ts)
	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("fork", "ActiveState-CLI/Python3", "--org", e2e.PersistentUsername),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Could not create the forked project", 30*time.Second)
	cp.Expect("The name 'Python3' is no longer available, it was used in a now deleted project.", 30*time.Second)
	cp.ExpectNotExitCode(0)
	suite.NotContains(cp.TrimmedSnapshot(), "Successfully forked project")
}

func TestForkIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ForkIntegrationTestSuite))
}
