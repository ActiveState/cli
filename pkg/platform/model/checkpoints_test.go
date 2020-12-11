package model_test

import (
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"

	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type CheckpointTestSuite struct {
	suite.Suite
	apiMock   *apiMock.Mock
	authMock  *authMock.Mock
	graphMock *graphMock.Mock
}

func (suite *CheckpointTestSuite) BeforeTest(suiteName, testName string) {
	suite.apiMock = apiMock.Init()
	suite.authMock = authMock.Init()
	suite.graphMock = graphMock.Init()
}

func (suite *CheckpointTestSuite) AfterTest(suiteName, testName string) {
	suite.apiMock.Close()
	suite.authMock.Close()
	suite.graphMock.Close()
}

func (suite *CheckpointTestSuite) TestGetCheckpoint() {
	suite.authMock.MockLoggedin()
	suite.graphMock.Checkpoint(graphMock.NoOptions)

	response, _, err := model.FetchCheckpointForCommit(strfmt.UUID("00010001-0001-0001-0001-000100010001"))
	suite.Require().NoError(err)
	suite.NotEmpty(response, "Returns checkpoints")
}

func (suite *CheckpointTestSuite) TestGetLanguages() {
	suite.authMock.MockLoggedin()
	suite.graphMock.Checkpoint(graphMock.NoOptions)

	response, err := model.FetchLanguagesForCommit(strfmt.UUID("00010001-0001-0001-0001-000100010001"))
	suite.Require().NoError(err)
	suite.NotEmpty(response, "Returns checkpoints")
	suite.Equal("Python", response[0].Name, "Returns Python")
}

func TestCheckpointSuite(t *testing.T) {
	suite.Run(t, new(CheckpointTestSuite))
}
