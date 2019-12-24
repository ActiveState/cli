package integration

import (
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/stretchr/testify/suite"
)

type ForkIntegrationTestSuite struct {
	integration.Suite
}

func (suite *ForkIntegrationTestSuite) TestFork_FailNameExists() {
	suite.LoginAsPersistentUser()
	suite.AppendEnv([]string{"ACTIVESTATE_CLI_DISABLE_RUNTIME=false"})

	suite.Spawn("fork", "ActiveState-CLI/Python3", "--org=cli-integration-tests")
	suite.Expect("Could not create the forked project", 30*time.Second)
	suite.Expect("The name 'Python3' is no longer available, it was used in a now deleted project.", 30*time.Second)
	suite.NotContains(suite.Output(), "Successfully forked project")
}

func TestForkIntegrationTestSuite(t *testing.T) {
	_ = suite.Run

	integration.RunParallel(t, new(ForkIntegrationTestSuite))
}
