package integration

import (
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/stretchr/testify/suite"
)

type ForkIntegrationTestSuite struct {
	integration.Suite
	username string
}

func (suite *ForkIntegrationTestSuite) TearDownTest() {
	suite.Suite.TearDownTest()
	suite.Spawn("auth", "logout")
	suite.Wait()
}

func (suite *ForkIntegrationTestSuite) TestFork_FailNameExists() {
	suite.LoginAsPersistentUser()
	suite.AppendEnv([]string{"ACTIVESTATE_CLI_DISABLE_RUNTIME=false"})

	suite.Spawn("fork", "ActiveState-CLI/Python3", "--org", integration.PersistentUsername)
	suite.Expect("Could not create the forked project", 30*time.Second)
	suite.Expect("The name 'Python3' is no longer available, it was used in a now deleted project.", 30*time.Second)
	suite.NotContains(suite.UnsyncedOutput(), "Successfully forked project")
}

func TestForkIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ForkIntegrationTestSuite))
}
