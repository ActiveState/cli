package model_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/locale"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
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

	orgs, err := model.FetchOrganizations()
	suite.NoError(err, "Fetched organizations")
	suite.Equal(1, len(orgs), "One organization fetched")
	suite.Equal("string", orgs[0].Name)
}

func (suite *OrganizationsTestSuite) TestOrganizations_FetchByURLName() {
	suite.apiMock.MockGetOrganization()

	org, err := model.FetchOrgByURLName("string")
	suite.NoError(err, "Fetched organizations")
	suite.Equal("string", org.URLname)
	suite.Equal("string", org.Name)
}

func (suite *OrganizationsTestSuite) TestOrganizations_FetchByURLName_404() {
	suite.apiMock.MockGetOrganization404()

	org, err := model.FetchOrgByURLName("string")
	suite.EqualError(err, locale.T("err_api_org_not_found"))
	suite.Nil(org)
}

func (suite *OrganizationsTestSuite) TestOrganization_FetchOrgMember() {
	suite.apiMock.MockGetOrganizationMembers()

	member, err := model.FetchOrgMember("string", "test")
	suite.NoError(err, "should be able to fetch member with no issue")
	suite.NotNil(member)
}

func (suite *OrganizationsTestSuite) TestOrganization_FetchOrgMember_404() {
	suite.apiMock.MockGetOrganizationMembers401()

	_, err := model.FetchOrgMember("string", "test")
	suite.EqualError(err, locale.T("err_api_not_authenticated"))
}

func (suite *OrganizationsTestSuite) TestOrganization_FetchOrgMember_NotFound() {
	suite.apiMock.MockGetOrganizationMembers()

	member, err := model.FetchOrgMember("string", "not_test")
	suite.EqualError(err, locale.T("err_api_member_not_found"))
	suite.Nil(member)
}

func (suite *OrganizationsTestSuite) TestOrganizations_InviteUserToOrg() {
	suite.apiMock.MockGetOrganization()

	suite.apiMock.MockInviteUserToOrg()

	invitation, err := model.InviteUserToOrg("string", true, "foo@bar.com")
	suite.NoError(err, "should have received invitation receipt")
	suite.Equal("foo@bar.com", invitation.Email)

}

func (suite *OrganizationsTestSuite) TestOrganizations_InviteUserToOrg404() {
	suite.apiMock.MockGetOrganization()

	suite.apiMock.MockInviteUserToOrg404()

	invitation, err := model.InviteUserToOrg("string", true, "string")
	suite.EqualError(err, locale.T("err_api_org_not_found"))
	suite.Nil(invitation)

}

func TestOrganizationsTestSuite(t *testing.T) {
	suite.Run(t, new(OrganizationsTestSuite))
}
