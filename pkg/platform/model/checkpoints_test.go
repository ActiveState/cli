package model_test

import (
	"testing"

	apiMock "github.com/ActiveState/cli/pkg/platform/api/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"
)

type CheckpointTestSuite struct {
	suite.Suite
	apiMock  *apiMock.Mock
	authMock *authMock.Mock
}

func (suite *CheckpointTestSuite) BeforeTest(suiteName, testName string) {
	suite.apiMock = apiMock.Init()
	suite.authMock = authMock.Init()
}

func (suite *CheckpointTestSuite) AfterTest(suiteName, testName string) {
	suite.apiMock.Close()
	suite.authMock.Close()
}

func (suite *CheckpointTestSuite) TestGetCheckpoint() {
	suite.authMock.MockLoggedin()
	suite.apiMock.MockVcsGetCheckpoint()

	response, fail := model.FetchCheckpointForCommit(strfmt.UUID("00010001-0001-0001-0001-000100010001"))
	suite.Require().NoError(fail.ToError())
	suite.NotEmpty(response, "Returns checkpoints")
}

func TestCheckpointSuite(t *testing.T) {
	suite.Run(t, new(CheckpointTestSuite))
}
