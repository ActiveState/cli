package model_test

import (
	"testing"

	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/stretchr/testify/suite"
)

type TiersTestSuite struct {
	suite.Suite
	apiMock  *apiMock.Mock
	authMock *authMock.Mock
}

func (suite *TiersTestSuite) BeforeTest(suiteName, testName string) {
	suite.apiMock = apiMock.Init()
	suite.authMock = authMock.Init()

	suite.authMock.MockLoggedin()
}

func (suite *TiersTestSuite) AfterTest(suiteName, testName string) {
	suite.apiMock.Close()
	suite.authMock.Close()
}

func (suite *TiersTestSuite) TestOrganizations_FetchAll() {
	suite.apiMock.MockGetOrganizations()

	orgs, fail := model.FetchOrganizations()
	suite.NoError(fail.ToError(), "Fetched organizations")
	suite.Equal(1, len(orgs), "One organization fetched")
	suite.Equal("string", orgs[0].Name)
}

func (suite *TiersTestSuite) TestOrganizations_FetchByURLName() {
	suite.apiMock.MockGetTiers()

	tiers, fail := model.FetchTiers()
	suite.NoError(fail.ToError(), "Fetched organizations")
	suite.Equal("string", tiers[0].Name)
}

func TestTiersTestSuite(t *testing.T) {
	suite.Run(t, new(TiersTestSuite))
}
