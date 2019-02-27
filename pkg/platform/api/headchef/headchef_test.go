package headchef_test

import (
	"net/url"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/wsmock"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/stretchr/testify/suite"
)

type RequestResult struct {
	BuildStarted         bool
	BuildCompleted       bool
	BuildCompletedResult headchef_models.BuildCompleted
	BuildFailed          bool
	BuildFailedMessage   string
	Failure              *failures.Failure
	Closed               bool
}

type HeadchefTestSuite struct {
	suite.Suite
	mock *wsmock.WsMock
}

func (suite *HeadchefTestSuite) BeforeTest(suiteName, testName string) {
	suite.mock = wsmock.Init()
}

func (suite *HeadchefTestSuite) AfterTest(suiteName, testName string) {
	go suite.mock.Close()
}

func (suite *HeadchefTestSuite) PerformRequest() *RequestResult {
	result := &RequestResult{}

	buildRecipe := &headchef_models.BuildRequestRecipe{}
	requester := &headchef_models.BuildRequestRequester{}
	buildRequest := &headchef_models.BuildRequest{Requester: requester, Recipe: buildRecipe}
	u, err := url.Parse("ws://example.org/ws")
	suite.Require().NoError(err)
	req := headchef.NewRequest(u, buildRequest, suite.mock.Dialer())

	req.OnBuildStarted(func() {
		result.BuildStarted = true
	})

	req.OnBuildCompleted(func(res headchef_models.BuildCompleted) {
		result.BuildCompleted = true
		result.BuildCompletedResult = res
	})

	req.OnBuildFailed(func(msg string) {
		result.BuildFailed = true
		result.BuildFailedMessage = msg
	})

	req.OnFailure(func(fail *failures.Failure) {
		result.Failure = fail
	})

	done := make(chan bool)

	req.OnClose(func() {
		result.Closed = true
		done <- true
		suite.mock.Close()
	})

	req.Start()
	<-done

	return result
}

func (suite *HeadchefTestSuite) TestSuccesfulBuild() {
	suite.mock.QueueResponse("build_started")
	suite.mock.QueueResponse("build_completed")

	result := suite.PerformRequest()

	suite.True(result.BuildStarted, "Fired OnBuildStarted")
	suite.True(result.BuildCompleted, "Fired OnBuildCompleted")
}

func (suite *HeadchefTestSuite) TestBuildFailure() {
	suite.mock.QueueResponse("build_started")
	suite.mock.QueueResponse("build_failed")

	result := suite.PerformRequest()

	suite.True(result.BuildStarted, "Fired OnBuildStarted")
	suite.True(result.BuildFailed, "Fired OnBuildFailed")
}

func (suite *HeadchefTestSuite) TestValidationFailure() {
	suite.mock.QueueResponse("validation_error")

	result := suite.PerformRequest()

	suite.NotNil(result.Failure, "Fired Validation Error")
	suite.True(result.Failure.Type.Matches(headchef.FailRequestValidation))
}

func (suite *HeadchefTestSuite) TestUnknownFailure() {
	suite.mock.QueueResponse("unknown_message")
	suite.mock.QueueResponse("build_completed")

	result := suite.PerformRequest()

	suite.True(result.BuildCompleted, "Comleted despite unknown message")
}

func (suite *HeadchefTestSuite) TestMalformedJsonFailure() {
	suite.mock.QueueResponse("malformed_json")
	suite.mock.QueueResponse("build_completed")

	result := suite.PerformRequest()

	suite.True(result.BuildCompleted, "Comleted despite malformed json in one of the messages")
}

func TestHeadchefTestSuite(t *testing.T) {
	suite.Run(t, new(HeadchefTestSuite))
}
