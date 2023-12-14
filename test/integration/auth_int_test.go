package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ActiveState/termtest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
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
	tagsuite.Suite
}

func (suite *AuthIntegrationTestSuite) TestAuth() {
	suite.OnlyRunForTags(tagsuite.Auth, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	user := ts.CreateNewUser()
	ts.LogoutUser()
	suite.interactiveLogin(ts, user.Username, user.Password)
	ts.LogoutUser()
	suite.loginFlags(ts, user.Username)
	suite.ensureLogout(ts)
}

func (suite *AuthIntegrationTestSuite) TestAuthToken() {
	suite.OnlyRunForTags(tagsuite.Auth, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn(tagsuite.Auth, "--token", e2e.PersistentToken, "-n")
	cp.Expect("logged in", termtest.OptExpectTimeout(40*time.Second))
	cp.ExpectExitCode(0)

	cp = ts.Spawn(tagsuite.Auth, "--non-interactive")
	cp.Expect("logged in", termtest.OptExpectTimeout(40*time.Second))
	cp.ExpectExitCode(0)

	ts.LogoutUser()
	suite.ensureLogout(ts)
}

func (suite *AuthIntegrationTestSuite) interactiveLogin(ts *e2e.Session, username, password string) {
	cp := ts.Spawn(tagsuite.Auth, "--prompt")
	cp.Expect("username:")
	cp.SendLine(username)
	cp.Expect("password:")
	cp.SendLine(password)
	cp.Expect("logged in")
	cp.ExpectExitCode(0)

	// still logged in?
	c2 := ts.Spawn(tagsuite.Auth)
	c2.Expect("You are logged in")
	c2.ExpectExitCode(0)
}

func (suite *AuthIntegrationTestSuite) loginFlags(ts *e2e.Session, username string) {
	cp := ts.Spawn(tagsuite.Auth, "--username", username, "--password", "bad-password")
	cp.Expect("You are not authorized, did you provide valid login credentials?")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *AuthIntegrationTestSuite) ensureLogout(ts *e2e.Session) {
	cp := ts.Spawn(tagsuite.Auth, "--prompt")
	cp.Expect("username:")
	cp.SendCtrlC()
}

type userJSON struct {
	Username string `json:"username,omitempty"`
}

func (suite *AuthIntegrationTestSuite) authOutput(method string) {
	data, err := json.Marshal(userJSON{
		Username: e2e.PersistentUsername,
	})
	suite.Require().NoError(err)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	expected := string(data)
	ts.LoginAsPersistentUser()
	cp := ts.Spawn(tagsuite.Auth, "--output", method)
	cp.Expect(`"}`)
	cp.ExpectExitCode(0)
	suite.Contains(cp.Output(), fmt.Sprintf("%s", string(expected)))
}

func (suite *AuthIntegrationTestSuite) TestAuth_JsonOutput() {
	suite.OnlyRunForTags(tagsuite.Auth, tagsuite.JSON)
	suite.authOutput("json")
}

func TestAuthIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AuthIntegrationTestSuite))
}
