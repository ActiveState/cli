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
	suite.Suite
}

func (suite *AuthIntegrationTestSuite) TestAuth() {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	username := ts.CreateNewUser()
	ts.LogoutUser()
	suite.interactiveLogin(ts, username)
	ts.LogoutUser()
	suite.loginFlags(ts, username)
}

func (suite *AuthIntegrationTestSuite) interactiveLogin(ts *e2e.Session, username string) {
	cp := ts.Spawn("auth")
	cp.Expect("username:")
	cp.SendLine(username)
	cp.Expect("password:")
	cp.SendLine(username)
	cp.Expect("successfully authenticated", 20*time.Second)
	cp.ExpectExitCode(0)

	// still logged in?
	c2 := ts.Spawn("auth")
	c2.Expect("You are logged in")
	c2.ExpectExitCode(0)
}

func (suite *AuthIntegrationTestSuite) loginFlags(ts *e2e.Session, username string) {
	cp := ts.Spawn("auth", "--username", username, "--password", "bad-password")
	cp.Expect("Authentication failed")
	cp.Expect("You are not authorized, did you provide valid login credentials?")
	cp.ExpectExitCode(1)
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

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	expected := string(data)
	ts.LoginAsPersistentUser()
	cp := ts.Spawn("auth", "--output", method)
	cp.Expect("false}")
	cp.ExpectExitCode(0)
	suite.Equal(fmt.Sprintf("%s", string(expected)), cp.TrimmedSnapshot())
}

func (suite *AuthIntegrationTestSuite) TestAuth_JsonOutput() {
	suite.authOutput("json")
}

func TestAuthIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AuthIntegrationTestSuite))
}
