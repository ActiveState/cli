package model_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/platform/api/client"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/stretchr/testify/suite"
)

type ProjectsTestSuite struct {
	suite.Suite
	apiMock  *apiMock.Mock
	authMock *authMock.Mock
}

func (suite *ProjectsTestSuite) BeforeTest(suiteName, testName string) {
	suite.apiMock = apiMock.Init()
	suite.authMock = authMock.Init()

	suite.authMock.MockLoggedin()
}

func (suite *ProjectsTestSuite) AfterTest(suiteName, testName string) {
	suite.apiMock.Close()
	suite.authMock.Close()
}

func (suite *ProjectsTestSuite) TestProjects_FetchByName() {
	project, fail := model.FetchProjectByName("example-org", "example-proj")
	suite.Require().NoError(fail.ToError(), "Fetched project")
	suite.Equal("example-proj", project.Name)
}

func (suite *ProjectsTestSuite) TestProjects_FetchByName_NotFound() {
	project, fail := model.FetchProjectByName("bad-org", "bad-proj")
	suite.EqualError(fail.ToError(), client.ErrNoValueAvailable.Error())
	suite.Nil(project)
}

func TestProjectsTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectsTestSuite))
}
