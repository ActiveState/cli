package integration

import (
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"strings"
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
	suite.Signup()
	suite.Logout()
	suite.Login()
}

func (suite *AuthIntegrationTestSuite) Signup() {
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

func (suite *AuthIntegrationTestSuite) Logout() {
	suite.Spawn("auth", "logout")
	defer suite.Stop()

	suite.Expect("You have been logged out")
	suite.Wait()
}

func (suite *AuthIntegrationTestSuite) Login() {
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

func (suite *AuthIntegrationTestSuite) TestAuth_JsonOutput() {
	type userJSON struct {
		Username        string `json:"username,omitempty"`
		URLName         string `json:"urlname,omitempty"`
		Tier            string `json:"tier,omitempty"`
		PrivateProjects bool   `json:"privateProjects"`
	}

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
	suite.Spawn("auth", "--json")
	if runtime.GOOS != "windows" {
		suite.Expect(expected)
	}
	suite.Wait()
	if runtime.GOOS == "windows" {
		// When the PTY reaches 80 characters it continues output on a new line.
		// On Windows this means both a carriage return and a new line. Windows
		// also picks up any spaces at the end of the console output, hence all
		// the cleaning we must do here.
		re := regexp.MustCompile("\r?\n")
		actual := strings.TrimSpace(re.ReplaceAllString(suite.Output(), ""))
		suite.Equal(expected, actual)
	}
}

func TestAuthIntegrationTestSuite(t *testing.T) {
	_ = suite.Run // vscode won't show test helpers unless I use this .. -.-

	//suite.Run(t, new(AuthIntegrationTestSuite))
	integration.RunParallel(t, new(AuthIntegrationTestSuite))
}
