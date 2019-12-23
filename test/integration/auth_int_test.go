package integration

import (
	"encoding/json"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
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
	integration.Suite
	username string
	password string
	email    string
}

func (suite *AuthIntegrationTestSuite) SetupTest() {
	suite.Suite.SetupTest()

	suite.username = fmt.Sprintf("user-%s", uid.String()[0:8])
	suite.password = suite.username
	suite.email = fmt.Sprintf("%s@test.tld", suite.username)
}

func (suite *AuthIntegrationTestSuite) TestAuth() {
	suite.signup()
	suite.logout()
	suite.login()
	suite.logout()
	suite.loginFlags()
}

func (suite *AuthIntegrationTestSuite) signup() {
	suite.Spawn("auth", "signup")
	defer suite.Stop()

	suite.Expect("username:")
	suite.SendLine(suite.username)
	suite.Expect("password:")
	suite.SendLine(suite.password)
	suite.Expect("again:")
	suite.SendLine(suite.password)
	suite.Expect("name:")
	suite.SendLine(suite.username)
	suite.Expect("email:")
	suite.SendLine(suite.email)
	suite.Expect("account has been registered", 20*time.Second)
	suite.Wait()
}

func (suite *AuthIntegrationTestSuite) logout() {
	suite.Spawn("auth", "logout")
	defer suite.Stop()

	suite.Expect("You have been logged out")
	suite.Wait()
}

func (suite *AuthIntegrationTestSuite) login() {
	suite.Spawn("auth")
	suite.Expect("username:")
	suite.SendLine(suite.username)
	suite.Expect("password:")
	suite.SendLine(suite.password)
	suite.Expect("successfully authenticated", 20*time.Second)
	suite.Wait()

	// still logged in?
	suite.Spawn("auth")
	suite.Expect("You are logged in")
	suite.Wait()
}

func (suite *AuthIntegrationTestSuite) loginFlags() {
	suite.Spawn("auth", "--username", suite.username, "--password", "bad-password")
	suite.Expect("Authentication failed")
	suite.Expect("You are not authorized, did you provide valid login credentials?")
	suite.Wait()
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
	suite.Spawn("auth", "--output", method)
	if runtime.GOOS != "windows" {
		suite.Expect(expected)
	}
	suite.Wait()
	suite.Equal(expected, suite.TrimOutput())
}

func (suite *AuthIntegrationTestSuite) TestAuth_JsonOutput() {
	suite.authOutput("json")
}

func (suite *AuthIntegrationTestSuite) TestAuthOutput_EditorV0() {
	suite.authOutput("editor.v0")
}

func (suite *AuthIntegrationTestSuite) TestAuth_EditorV0() {
	user := userJSON{
		Username:        "cli-integration-tests",
		URLName:         "cli-integration-tests",
		Tier:            "free",
		PrivateProjects: false,
	}
	data, err := json.Marshal(user)
	suite.Require().NoError(err)
	expected := string(data)

	suite.Spawn("auth", "--username", integration.PersistentUsername, "--password", integration.PersistentPassword, "--output", "editor.v0")
	suite.Expect(expected)
	suite.Wait()
}

func TestAuthIntegrationTestSuite(t *testing.T) {
	_ = suite.Run // vscode won't show test helpers unless I use this .. -.-

	//suite.Run(t, new(AuthIntegrationTestSuite))
	integration.RunParallel(t, new(AuthIntegrationTestSuite))
}
