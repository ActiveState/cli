package deprecation_test

import (
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/ActiveState/cli/internal/deprecation"
	depMock "github.com/ActiveState/cli/internal/deprecation/mock"

	"github.com/stretchr/testify/suite"
)

type DeprecationTestSuite struct {
	suite.Suite
	mock *depMock.Mock
}

func (suite *DeprecationTestSuite) BeforeTest(suiteName, testName string) {
	suite.mock = depMock.Init()
	viper.Reset()
}

func (suite *DeprecationTestSuite) AfterTest(suiteName, testName string) {
	suite.mock.Close()
}

func (suite *DeprecationTestSuite) TestDeprecation() {
	suite.mock.MockExpired()

	deprecated, fail := deprecation.CheckVersionNumber("0.11.18")
	suite.Require().NoError(fail)
	suite.NotNil(deprecated, "Returns deprecation info")
	suite.Equal("999.0.0", deprecated.Version, "Fails on the most recent applicable version")
	suite.True(deprecated.DateReached, "Deprecation date has been reached")
}

func (suite *DeprecationTestSuite) TestDeprecationHandlesZeroed() {
	suite.mock.MockExpired()

	deprecated, fail := deprecation.CheckVersionNumber("0.0.0")
	suite.Require().NoError(fail)
	suite.Nil(deprecated, "Returns no deprecation info")
}

func (suite *DeprecationTestSuite) TestDeprecationFuture() {
	suite.mock.MockDeprecated()

	deprecated, fail := deprecation.CheckVersionNumber("0.11.18")
	suite.Require().NoError(fail)
	suite.NotNil(deprecated, "Returns deprecation info")
	suite.False(deprecated.DateReached, "Deprecation date has not been reached")
}

func (suite *DeprecationTestSuite) TestDeprecationTimeout() {
	suite.mock.MockExpiredTimed(deprecation.DefaultTimeout + time.Second)

	_, fail := deprecation.CheckVersionNumber("0.11.18")
	suite.Require().NoError(fail) // timeouts should be handled gracefully inside the package
}

func TestDeprecationTestSuite(t *testing.T) {
	suite.Run(t, new(DeprecationTestSuite))
}
