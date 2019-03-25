package model_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/locale"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mock"
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
	suite.apiMock.MockGetProject()

	project, fail := model.FetchProjectByName("string", "string")
	suite.NoError(fail.ToError(), "Fetched project")
	suite.Equal("string", project.Name)
}

func (suite *ProjectsTestSuite) TestProjects_FetchByName_NotFound() {
	suite.apiMock.MockGetProject404()

	project, fail := model.FetchProjectByName("string", "string")
	suite.EqualError(fail.ToError(), locale.T("err_api_project_not_found"))
	suite.Nil(project)
}

func TestProjectsTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectsTestSuite))
}
