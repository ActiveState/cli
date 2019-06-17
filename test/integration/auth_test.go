package integration_test

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/test/integration/expect"
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

type AuthTestSuite struct {
	Suite
	username string
	password string
	email    string
}

func (suite *AuthTestSuite) SetupTest() {
	suite.Suite.SetupTest()

	suite.username = fmt.Sprintf("user-%s", uid.String()[0:8])
	suite.password = suite.username
	suite.email = fmt.Sprintf("%s@test.tld", suite.username)
}

func (suite *AuthTestSuite) TestAuth() {
	suite.Signup()
	suite.Logout()
	suite.Login()
}

func (suite *AuthTestSuite) Signup() {
	suite.Spawn("auth", "signup")
	defer suite.Stop()

	suite.Expect("username:")
	suite.Send(suite.username)
	suite.Expect("password:")
	suite.Send(suite.password)
	suite.Expect("again:")
	suite.Send(suite.password)
	suite.Expect("name:")
	suite.Send(suite.username)
	suite.Expect("email:")
	suite.Send(suite.email)
	suite.Expect("account has been registered")
	suite.Wait()
}

func (suite *AuthTestSuite) Logout() {
	suite.Spawn("auth", "logout")
	defer suite.Stop()

	suite.Expect("You have been logged out")
	suite.Wait()
}

func (suite *AuthTestSuite) Login() {
	suite.Spawn("auth")
	suite.Expect("username:")
	suite.Send(suite.username)
	suite.Expect("password:")
	suite.Send(suite.password)
	suite.Expect("succesfully authenticated")
	suite.Wait()

	// still logged in?
	suite.Spawn("auth")
	suite.Expect("You are logged in")
	suite.Wait()
}

func TestAuthTestSuite(t *testing.T) {
	_ = suite.Run // vscode won't show test helpers unless I use this .. -.-

	//suite.Run(t, new(AuthTestSuite))
	expect.RunParallel(t, new(AuthTestSuite))
}
