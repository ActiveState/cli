package analytics

import (
	"testing"

	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/stretchr/testify/suite"
)

const CatTest = "tests"

type AnalyticsTestSuite struct {
	suite.Suite

	authMock *authMock.Mock
}

func (suite *AnalyticsTestSuite) BeforeTest(suiteName, testName string) {
	suite.authMock = authMock.Init()
	suite.authMock.MockLoggedin()
}

func (suite *AnalyticsTestSuite) TestSetup() {
	setup()
	suite.Require().NotNil(client, "Client is set")
}

func (suite *AnalyticsTestSuite) TestEvent() {
	err := event(CatTest, "TestEvent")
	suite.Require().NoError(err, "Should send event without causing an error")
}

func (suite *AnalyticsTestSuite) TestEventWithValue() {
	err := eventWithValue(CatTest, "TestEventWithValue", 1)
	suite.Require().NoError(err, "Should send event with value without causing an error")
}

func TestAnalyticsTestSuite(t *testing.T) {
	suite.Run(t, new(AnalyticsTestSuite))
}
