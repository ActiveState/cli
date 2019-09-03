package model_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/locale"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/stretchr/testify/suite"
)

type OrganizationsTestSuite struct {
	suite.Suite
	apiMock  *apiMock.Mock
	authMock *authMock.Mock
}

func (suite *OrganizationsTestSuite) BeforeTest(suiteName, testName string) {
	suite.apiMock = apiMock.Init()
	suite.authMock = authMock.Init()

	suite.authMock.MockLoggedin()
}

func (suite *OrganizationsTestSuite) AfterTest(suiteName, testName string) {
	suite.apiMock.Close()
	suite.authMock.Close()
}

func (suite *OrganizationsTestSuite) TestOrganizations_FetchAll() {
	suite.apiMock.MockGetOrganizations()

	orgs, fail := model.FetchOrganizations()
	suite.NoError(fail.ToError(), "Fetched organizations")
	suite.Equal(1, len(orgs), "One organization fetched")
	suite.Equal("string", orgs[0].Name)
}

func (suite *OrganizationsTestSuite) TestOrganizations_FetchByURLName() {
	suite.apiMock.MockGetOrganization()

	org, fail := model.FetchOrgByURLName("string")
	suite.NoError(fail.ToError(), "Fetched organizations")
	suite.Equal("string", org.Urlname)
	suite.Equal("string", org.Name)
}

func (suite *OrganizationsTestSuite) TestOrganizations_FetchByURLName_404() {
	suite.apiMock.MockGetOrganization404()

	org, fail := model.FetchOrgByURLName("string")
	suite.EqualError(fail, locale.T("err_api_org_not_found"))
	suite.Nil(org)
}

func (suite *OrganizationsTestSuite) TestOrganizations_InviteUserToOrg() {
	suite.apiMock.MockGetOrganization()

	org, fail := model.FetchOrgByURLName("string")
	suite.NoError(fail.ToError(), "should have received org")

	suite.apiMock.MockInviteUserToOrg()

	invitation, fail := model.InviteUserToOrg(org, true, "string")
	suite.NoError(fail.ToError(), "should have received invitation receipt")
	suite.Equal("string", invitation.Email)
	suite.Equal(org, invitation.Organization)

}

func (suite *OrganizationsTestSuite) TestOrganizations_InviteUserToOrg404() {
	suite.apiMock.MockGetOrganization()

	org, fail := model.FetchOrgByURLName("string")
	suite.NoError(fail.ToError(), "should have received org")

	suite.apiMock.MockInviteUserToOrg404()

	invitation, fail := model.InviteUserToOrg(org, true, "string")
	suite.EqualError(fail, locale.T("err_api_org_not_found"))
	suite.Nil(invitation)

}

func TestOrganizationsTestSuite(t *testing.T) {
	suite.Run(t, new(OrganizationsTestSuite))
}
