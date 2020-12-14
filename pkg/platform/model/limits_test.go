package model_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/locale"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
)

type LimitsTestSuite struct {
	suite.Suite
	apiMock  *apiMock.Mock
	authMock *authMock.Mock
}

func (suite *LimitsTestSuite) BeforeTest(suiteName, testName string) {
	suite.apiMock = apiMock.Init()
	suite.authMock = authMock.Init()

	suite.authMock.MockLoggedin()
}

func (suite *LimitsTestSuite) AfterTest(suiteName, testName string) {
	suite.apiMock.Close()
	suite.authMock.Close()
}

func (suite *LimitsTestSuite) TestLimits_FetchLimits() {
	suite.apiMock.MockGetOrganizationLimits()

	limits, err := model.FetchOrganizationLimits("string")
	suite.NoError(err, "Fetched organization limits")
	suite.NotNil(limits, "expected to retrieve limits")
	suite.Equal(50, limits.NodesLimit)
	suite.Equal(100, limits.UsersLimit)
}

func (suite *OrganizationsTestSuite) TestLimits_FetchLimits_404() {
	suite.apiMock.MockGetOrganizationLimits404()

	_, err := model.FetchOrganizationLimits("string")
	suite.EqualError(err, locale.T("err_api_org_not_found"))
}

func TestLimitsTestSuite(t *testing.T) {
	suite.Run(t, new(OrganizationsTestSuite))
}
