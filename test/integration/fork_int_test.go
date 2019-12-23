package integration

import (
	"encoding/json"
	"fmt"
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

	suite.Spawn("fork", "ActiveState-CLI/Python3")
	suite.Expect("Could not create the forked project", 30*time.Second)
	suite.Expect("The name 'Python3' is no longer available, it was used in a now deleted project.", 30*time.Second)
	suite.NotContains(suite.Output(), "Successfully forked project")
}

func (suite *ForkIntegrationTestSuite) createNewUser() {
	suite.username = fmt.Sprintf("user-%s", uid.String()[0:8])
	password := suite.username
	email := fmt.Sprintf("%s@test.tld", suite.username)

	suite.Spawn("auth", "signup")
	suite.Expect("username:")
	suite.SendLine(suite.username)
	suite.Expect("password:")
	suite.SendLine(password)
	suite.Expect("again:")
	suite.SendLine(password)
	suite.Expect("name:")
	suite.SendLine(suite.username)
	suite.Expect("email:")
	suite.SendLine(email)
	suite.Expect("account has been registered", 20*time.Second)
	suite.Wait()
}

func (suite *ForkIntegrationTestSuite) TestFork_EditorV0() {
	suite.createNewUser()

	results := struct {
		Result map[string]string `json:"result,omitempty"`
	}{
		map[string]string{
			"NewName":       "Test-Python3",
			"NewOwner":      suite.username,
			"OriginalName":  "Python3",
			"OriginalOwner": "ActiveState-CLI",
		},
	}
	expected, err := json.Marshal(results)
	suite.Require().NoError(err)

	suite.Spawn("fork", "ActiveState-CLI/Python3", "--name", "Test-Python3", "--org", suite.username, "--output", "editor.v0")
	suite.Wait()
	suite.Equal(string(expected), suite.TrimOutput())
}

func TestForkIntegrationTestSuite(t *testing.T) {
	_ = suite.Run

	integration.RunParallel(t, new(ForkIntegrationTestSuite))
}
