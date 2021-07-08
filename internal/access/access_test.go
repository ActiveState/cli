package access

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
)

type SecretsTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
	platformMock  *httpmock.HTTPMock
	authMock      *authMock.Mock
}

func (suite *SecretsTestSuite) BeforeTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")
	err = os.Chdir(filepath.Join(root, "internal", "access", "testdata"))
	suite.Require().NoError(err, "Should chdir")

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())

	suite.authMock = authMock.Init()
	suite.authMock.MockLoggedin()
}

func (suite *SecretsTestSuite) AfterTest(suiteName, testName string) {
	cfg, err := config.New()
	suite.Require().NoError(err)
	defer cfg.Close()
	osutil.RemoveConfigFile(cfg.ConfigPath(), constants.KeypairLocalFileName+".key")
	httpmock.DeActivate()
	suite.authMock.Close()
}

func (suite *SecretsTestSuite) TestSecretsNoAccess() {
	suite.platformMock.RegisterWithCode("GET", "/organizations/AccessOrg", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/AccessOrg/members", 200)

	hasAccess, err := Secrets("AccessOrg")
	suite.Require().NoError(err, "unexepected error checking for secret access")
	suite.Equal(false, hasAccess, "should not have access to secrets")
}

func TestSecretsTestSuite(t *testing.T) {
	suite.Run(t, new(SecretsTestSuite))
}
