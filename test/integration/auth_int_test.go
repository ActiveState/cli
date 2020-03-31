package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
)

var uid uuid.UUID

func init() {
	// Because of our parallel test running tweak normal suite operations are limited and we have to set up shared resources
	// at a higher level. We can better address this in the long run but for now this'll have to do.
	var err error
	uid, err = uuid.NewRandom()
	if err != nil {
		panic(fmt.Sprintf("Could not generate uuid, error: %v", err))
	}
}

type AuthIntegrationTestSuite struct {
	e2e.Suite
}

func (suite *AuthIntegrationTestSuite) TestAuth() {
	username := suite.CreateNewUser()
	suite.LogoutUser()
	suite.interactiveLogin(username)
	suite.LogoutUser()
	suite.loginFlags(username)
}

func (suite *AuthIntegrationTestSuite) interactiveLogin(username string) {
	cp := suite.Spawn("auth")
	defer cp.Close()
	cp.Expect("username:")
	cp.SendLine(username)
	cp.Expect("password:")
	cp.SendLine(username)
	cp.Expect("successfully authenticated", 20*time.Second)
	cp.ExpectExitCode(0)

	// still logged in?
	cp = suite.Spawn("auth")
	cp.Expect("You are logged in")
	cp.ExpectExitCode(0)
}

func (suite *AuthIntegrationTestSuite) loginFlags(username string) {
	cp := suite.Spawn("auth", "--username", username, "--password", "bad-password")
	cp.Expect("Authentication failed")
	cp.Expect("You are not authorized, did you provide valid login credentials?")
	cp.ExpectExitCode(0)
}

type userJSON struct {
	Username        string `json:"username,omitempty"`
	URLName         string `json:"urlname,omitempty"`
	Tier            string `json:"tier,omitempty"`
	PrivateProjects bool   `json:"privateProjects"`
}

func (suite *AuthIntegrationTestSuite) authOutput(method string) {
	user := userJSON{
		Username:        "cli-integration-tests",
		URLName:         "cli-integration-tests",
		Tier:            "free",
		PrivateProjects: false,
	}
	data, err := json.Marshal(user)
	suite.Require().NoError(err)

	expected := string(data)
	suite.LoginAsPersistentUser()
	cp := suite.Spawn("auth", "--output", method)
	cp.Expect("false}")
	suite.Equal(fmt.Sprintf("%s", string(expected)), cp.TrimmedSnapshot())
}

func (suite *AuthIntegrationTestSuite) TestAuth_JsonOutput() {
	suite.authOutput("json")
}

func (suite *AuthIntegrationTestSuite) TestAuthOutput_EditorV0() {
	suite.authOutput("editor.v0")
}

func (suite *AuthIntegrationTestSuite) TestAuth_EditorV0() {
	user := userJSON{
		Username: "cli-integration-tests",
		URLName:  "cli-integration-tests",
		Tier:     "free",
	}
	data, err := json.Marshal(user)
	suite.Require().NoError(err)
	expected := string(data)

	cp := suite.Spawn("auth", "--username", e2e.PersistentUsername, "--password", e2e.PersistentPassword, "--output", "editor.v0")
	cp.Expect(`"privateProjects":false}`)
	cp.ExpectExitCode(0)
	suite.Equal(fmt.Sprintf("%s", string(expected)), cp.TrimmedSnapshot())
}

func TestAuthIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AuthIntegrationTestSuite))
}
