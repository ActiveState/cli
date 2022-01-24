package project_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type SecretsExpanderTestSuite struct {
	suite.Suite

	projectFile *projectfile.Project
	project     *project.Project
	cfg         keypairs.Configurable

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
	graphMock     *mock.Mock
}

func loadSecretsProject() (*projectfile.Project, error) {
	pjfile := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/SecretOrg/SecretProject?commitID=00010001-0001-0001-0001-000100010001"
`)

	err := yaml.Unmarshal([]byte(contents), pjfile)
	if err != nil {
		return nil, err
	}
	err = pjfile.Init()
	if err != nil {
		return nil, err
	}

	return pjfile, nil
}

func (suite *SecretsExpanderTestSuite) BeforeTest(suiteName, testName string) {
	locale.Set("en-US")

	projectFile, err := loadSecretsProject()
	suite.Require().Nil(err, "Unmarshalled project YAML")
	projectFile.Persist()
	suite.projectFile = projectFile
	suite.project = project.Get()

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())

	suite.platformMock.Register("POST", "/login")
	suite.platformMock.Register("GET", "/organizations/SecretOrg/members")
	authentication.LegacyGet().AuthenticateWithToken("")

	suite.graphMock = mock.Init()
	suite.graphMock.ProjectByOrgAndName(mock.NoOptions)

	suite.cfg, err = config.New()
	suite.Require().NoError(err)
}

func (suite *SecretsExpanderTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	projectfile.Reset()
	osutil.RemoveConfigFile(suite.cfg.ConfigPath(), constants.KeypairLocalFileName+".key")
	suite.graphMock.Close()
	suite.Require().NoError(suite.cfg.Close())
}

func (suite *SecretsExpanderTestSuite) prepareWorkingExpander() project.ExpanderFunc {
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg", 200)

	osutil.CopyTestFileToConfigDir(suite.cfg.ConfigPath(), "self-private.key", constants.KeypairLocalFileName+".key", 0600)

	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010002/user_secrets", 200)
	return project.NewSecretQuietExpander(suite.secretsClient, suite.cfg)
}

func (suite *SecretsExpanderTestSuite) assertExpansionFailure(secretName string) {
	value, err := suite.prepareWorkingExpander()("", project.ProjectCategory, secretName, false, suite.project)
	suite.Require().Error(err)
	suite.Zero(value)
}

func (suite *SecretsExpanderTestSuite) assertExpansionSuccess(secretName string, expectedExpansionValue string, isUser bool) {
	category := project.ProjectCategory
	if isUser {
		category = project.UserCategory
	}
	value, err := suite.prepareWorkingExpander()("", category, secretName, false, suite.project)
	suite.Equal(expectedExpansionValue, value)
	suite.Nil(err)
}

func (suite *SecretsExpanderTestSuite) TestKeypairNotFound() {
	expanderFn := project.NewSecretQuietExpander(suite.secretsClient, suite.cfg)
	value, err := expanderFn("", project.ProjectCategory, "undefined-secret", false, suite.project)
	suite.Error(err)
	suite.Zero(value)
}

func (suite *SecretsExpanderTestSuite) TestProjectSecret() {
	suite.assertExpansionSuccess("proj-secret", "proj-value", false)
}

func (suite *SecretsExpanderTestSuite) TestUserSecret() {
	suite.assertExpansionSuccess("user-proj-secret", "user-proj-value", true)
}

func Test_SecretsExpander_TestSuite(t *testing.T) {
	suite.Run(t, new(SecretsExpanderTestSuite))
}
