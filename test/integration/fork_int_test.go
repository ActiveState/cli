package integration

import (
	"encoding/json"
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
	suite.NotContains(suite.Output(), "Successfully forked project")
}

func (suite *ForkIntegrationTestSuite) TestFork_EditorV0() {
	username := suite.CreateNewUser()

	results := struct {
		Result map[string]string `json:"result,omitempty"`
	}{
		map[string]string{
			"NewName":       "Test-Python3",
			"NewOwner":      username,
			"OriginalName":  "Python3",
			"OriginalOwner": "ActiveState-CLI",
		},
	}
	expected, err := json.Marshal(results)
	suite.Require().NoError(err)

	suite.Spawn("fork", "ActiveState-CLI/Python3", "--name", "Test-Python3", "--org", username, "--output", "editor.v0")
	suite.Expect(string(expected))
}

func TestForkIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ForkIntegrationTestSuite))
}
