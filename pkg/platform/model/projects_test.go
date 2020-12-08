package model_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type ProjectsTestSuite struct {
	suite.Suite
	apiMock   *apiMock.Mock
	authMock  *authMock.Mock
	graphMock *graphMock.Mock
}

func (suite *ProjectsTestSuite) BeforeTest(suiteName, testName string) {
	suite.apiMock = apiMock.Init()
	suite.authMock = authMock.Init()
	suite.graphMock = graphMock.Init()

	suite.authMock.MockLoggedin()
	suite.graphMock.ProjectByOrgAndName(graphMock.NoOptions)
}

func (suite *ProjectsTestSuite) AfterTest(suiteName, testName string) {
	suite.apiMock.Close()
	suite.authMock.Close()
	suite.graphMock.Close()
}

func (suite *ProjectsTestSuite) TestProjects_FetchByName() {
	project, fail := model.FetchProjectByName("string", "string")
	suite.Require().NoError(fail, "Fetched project")
	suite.Equal("string", project.Name)
}

func (suite *ProjectsTestSuite) TestProjects_FetchByName_NotFound() {
	suite.graphMock.Reset()
	suite.graphMock.NoProjects(graphMock.NoOptions)
	project, fail := model.FetchProjectByName("bad-org", "bad-proj")
	suite.Require().Error(fail)
	suite.Equal(fail.Type.Name, model.FailProjectNotFound.Name)
	suite.Nil(project)
}

func TestProjectsTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectsTestSuite))
}
