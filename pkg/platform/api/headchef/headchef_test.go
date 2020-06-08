package headchef_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/mock"
	headchefMock "github.com/ActiveState/cli/pkg/platform/api/headchef/mock"
)

var maxWait = time.Second * 2

type HeadchefTestSuite struct {
	suite.Suite
	mock *headchefMock.Mock
}

func (suite *HeadchefTestSuite) BeforeTest(suiteName, testName string) {
	suite.mock = headchefMock.Init()
}

func (suite *HeadchefTestSuite) AfterTest(suiteName, testName string) {
	suite.mock.Close()
}

func (suite *HeadchefTestSuite) SendRequest(rt headchefMock.ResponseType) *headchef.BuildStatus {
	suite.mock.MockBuilds(rt)

	client := headchef.NewClient(api.GetServiceURL(api.ServiceHeadChef))
	buildRequest := &headchef_models.V1BuildRequest{
		Requester: &headchef_models.V1Requester{},
	}
	return client.RequestBuild(buildRequest)
}

func (suite *HeadchefTestSuite) TestBuildStarted() {
	status := suite.SendRequest(mock.Started)

	select {
	case _, ok := <-status.Started:
		suite.True(ok, "started channel must not be closed")
	case <-time.After(maxWait):
		suite.FailNow("started not received")
	}
}

func (suite *HeadchefTestSuite) TestBuildFailed() {
	status := suite.SendRequest(mock.Failed)

	select {
	case msg, ok := <-status.Failed:
		suite.True(ok, "failed channel must not be closed")
		suite.NotEmpty(msg, "failed build requires message")
	case <-time.After(maxWait):
		suite.FailNow("failed not received")
	}
}

func (suite *HeadchefTestSuite) TestBuildCompleted() {
	status := suite.SendRequest(mock.Completed)

	select {
	case statusResp, ok := <-status.Completed:
		suite.True(ok, "completed channel must not be closed")
		suite.NotNil(statusResp, "completed status response must not be nil")
		suite.NotEmpty(statusResp.Artifacts, "completed artifacts must not be empty")

	case <-time.After(maxWait):
		suite.FailNow("completed not received")
	}
}

func (suite *HeadchefTestSuite) TestBuildRunFail() {
	status := suite.SendRequest(mock.RunFail)

	select {
	case fail, ok := <-status.RunFail:
		suite.True(ok, "runfail channel must not be closed")
		suite.NotNil(fail, "runfail failure must not be nil")

		failMatches := fail.Type.Matches(headchef.FailBuildReqErrorResp)
		suite.True(failMatches, "runfail failure must be correct type")

	case <-time.After(maxWait):
		suite.FailNow("runfail not received")
	}
}

func (suite *HeadchefTestSuite) TestBuildRunFailMalformed() {
	status := suite.SendRequest(mock.RunFailMalformed)

	select {
	case fail, ok := <-status.RunFail:
		suite.True(ok, "runfail channel must not be closed")
		suite.NotNil(fail, "runfail failure must not be nil")

		failMatches := fail.Type.Matches(headchef.FailBuildCreatedUnknownType)
		suite.True(failMatches, "runfail failure must be correct type")

	case <-time.After(maxWait):
		suite.FailNow("runfail not received")
	}
}

func TestHeadchefTestSuite(t *testing.T) {
	suite.Run(t, new(HeadchefTestSuite))
}
