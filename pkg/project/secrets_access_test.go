package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
)

type SecretsAccessTestSuite struct {
	suite.Suite

	projectFile *projectfile.Project

	secretsClient *secretsapi.Client
	platformMock  *httpmock.HTTPMock
	authMock      *authMock.Mock
	expander      *SecretExpander
}

func (suite *SecretsAccessTestSuite) BeforeTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")
	err = os.Chdir(filepath.Join(root, "pkg", "project", "testdata", "access"))
	suite.Require().NoError(err, "Should chdir")

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())

	suite.authMock = authMock.Init()
	suite.authMock.MockLoggedin()

	suite.expander = NewSecretExpander(suite.secretsClient, nil)
	suite.expander.project = Get()
}

func (suite *SecretsAccessTestSuite) TestFindSecretNoAccess() {
	suite.platformMock.RegisterWithCode("GET", "/organizations/AccessOrg", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/AccessOrg/members", 200)

	_, fail := suite.expander.FindSecret("does.not.matter", false)
	suite.Require().Error(fail.ToError(), "should get an error when we do not have access")
	suite.Equal(fail.Error(), locale.Tr("secrets_expand_err_no_access", "AccessOrg"))
}

func (suite *SecretsAccessTestSuite) TestFindSecretAccessError() {
	suite.platformMock.RegisterWithCode("GET", "/organizations/AccessOrg", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/AccessOrg/members", 401)

	_, fail := suite.expander.FindSecret("does.not.matter", false)
	suite.Require().Error(fail.ToError(), "should get an error when not authorized")
}

func TestSecretsAccessTestSuite(t *testing.T) {
	suite.Run(t, new(SecretsAccessTestSuite))
}
