package auth_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/ActiveState/cli/pkg/expect"
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
	suite.Expect("account has been registered")
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
	suite.Expect("succesfully authenticated")
	suite.Wait()

	// still logged in?
	suite.Spawn("auth")
	suite.Expect("You are logged in")
	suite.Wait()
}

func TestAuthIntegrationTestSuite(t *testing.T) {
	_ = suite.Run // vscode won't show test helpers unless I use this .. -.-

	//suite.Run(t, new(AuthIntegrationTestSuite))
	expect.RunParallel(t, new(AuthIntegrationTestSuite))
}
