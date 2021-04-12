package deprecation_test

import (
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/deprecation"
	depMock "github.com/ActiveState/cli/internal/deprecation/mock"

	"github.com/stretchr/testify/suite"
)

type DeprecationTestSuite struct {
	suite.Suite
	mock *depMock.Mock
	cfg  deprecation.Configurable
}

func (suite *DeprecationTestSuite) BeforeTest(suiteName, testName string) {
	suite.mock = depMock.Init()
	var err error
	suite.cfg, err = config.Get()
	suite.Require().NoError(err)
}

func (suite *DeprecationTestSuite) AfterTest(suiteName, testName string) {
	suite.mock.Close()
}

func (suite *DeprecationTestSuite) xTestDeprecation() {
	suite.mock.MockExpired()

	deprecated, err := deprecation.CheckVersionNumber(suite.cfg, "0.11.18")
	suite.Require().NoError(err)
	suite.NotNil(deprecated, "Returns deprecation info")
	suite.Equal("999.0.0", deprecated.Version, "Fails on the most recent applicable version")
	suite.True(deprecated.DateReached, "Deprecation date has been reached")
}

func (suite *DeprecationTestSuite) xTestDeprecationHandlesZeroed() {
	suite.mock.MockExpired()

	deprecated, err := deprecation.CheckVersionNumber(suite.cfg, "0.0.0")
	suite.Require().NoError(err)
	suite.Nil(deprecated, "Returns no deprecation info")
}

func (suite *DeprecationTestSuite) xTestDeprecationFuture() {
	suite.mock.MockDeprecated()

	deprecated, err := deprecation.CheckVersionNumber(suite.cfg, "0.11.18")
	suite.Require().NoError(err)
	suite.NotNil(deprecated, "Returns deprecation info")
	suite.False(deprecated.DateReached, "Deprecation date has not been reached")
}

func (suite *DeprecationTestSuite) TestDeprecationTimeout() {
	suite.mock.MockExpiredTimed(deprecation.DefaultTimeout + time.Second)

	_, err := deprecation.CheckVersionNumber(suite.cfg, "0.11.18")
	suite.Require().NoError(err) // timeouts should be handled gracefully inside the package
}

func TestDeprecationTestSuite(t *testing.T) {
	suite.Run(t, new(DeprecationTestSuite))
}
