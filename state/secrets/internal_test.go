package secrets

import (
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type GetSecretTestSuite struct {
	suite.Suite
}

func (suite *GetSecretTestSuite) BeforeTest(suiteName, testName string) {
	p := &projectfile.Project{}
	p.Persist()
}

func (suite *GetSecretTestSuite) AfterTest(suiteName, testName string) {
	projectfile.Reset()
}

func (suite *GetSecretTestSuite) TestGetUserSecret() {
	secret, fail := getSecret("user.foo")
	suite.Require().NoError(fail.ToError())
	suite.True(secret.IsUser(), "Is user secret")
}

func (suite *GetSecretTestSuite) TestGetProjectSecret() {
	secret, fail := getSecret("project.foo")
	suite.Require().NoError(fail.ToError())
	suite.True(secret.IsProject(), "Is project secret")
}

func (suite *GetSecretTestSuite) TestGetSecretFailTooManyDots() {
	_, fail := getSecret("project.toomanydots.foo")
	suite.Require().Error(fail.ToError())
	suite.Equal(failures.FailUserInput.Name, fail.Type.Name)
}

func (suite *GetSecretTestSuite) TestGetSecretFailScope() {
	_, fail := getSecret("invalid.foo")
	suite.Require().Error(fail.ToError())
	suite.Equal(failures.FailInput.Name, fail.Type.Name)
}
