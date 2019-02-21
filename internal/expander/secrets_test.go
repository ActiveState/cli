package expander_test

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/authentication"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/expander"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
	yaml "gopkg.in/yaml.v2"
)

type SecretsExpanderTestSuite struct {
	suite.Suite

	projectFile *projectfile.Project

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
}

func loadSecretsProject() (*projectfile.Project, error) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
name: SecretProject
owner: SecretOrg
variables:
  - name: undefined-secret
    value:
      pullfrom: organization
      share: organization
  - name: org-secret
    value:
      pullfrom: organization
      share: organization
  - name: proj-secret
    value:
      pullfrom: project
      share: organization
  - name: user-secret
    value:
      pullfrom: organization
  - name: user-proj-secret
    value:
      pullfrom: project
  - name: org-secret-with-proj-value
    value:
      pullfrom: organization
      share: organization
  - name: proj-secret-with-user-value
    value:
      pullfrom: project
  - name: user-secret-with-user-proj-value
    value:
      pullfrom: organization
  - name: proj-secret-only-org-available
    value:
      pullfrom: project
      share: organization
  - name: user-secret-only-proj-available
    value:
      pullfrom: project
  - name: user-proj-secret-only-user-available
    value:
      pullfrom: project
  - name: bad-base64-encoded-secret
    value:
      pullfrom: organization
      share: organization
  - name: invalid-encryption-secret
    value:
      pullfrom: organization
      share: organization
`)

	err := yaml.Unmarshal([]byte(contents), project)
	if err != nil {
		return nil, err
	}

	return project, project.Parse()
}

func (suite *SecretsExpanderTestSuite) BeforeTest(suiteName, testName string) {
	locale.Set("en-US")
	failures.ResetHandled()

	projectFile, err := loadSecretsProject()
	suite.Require().Nil(err, "Unmarshalled project YAML")
	projectFile.Persist()
	suite.projectFile = projectFile

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String())

	suite.platformMock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")
}

func (suite *SecretsExpanderTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	projectfile.Reset()
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
}

func (suite *SecretsExpanderTestSuite) prepareWorkingExpander() expander.Func {
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg/projects/SecretProject", 200)

	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010002/user_secrets", 200)
	return expander.NewVarExpander(suite.secretsClient)
}

func (suite *SecretsExpanderTestSuite) assertExpansionFailure(secretName string, expectedFailureType *failures.FailureType) {
	value, failure := suite.prepareWorkingExpander()(secretName, suite.projectFile)
	suite.True(failure.Type.Matches(expectedFailureType), "unexpected failure type")
	suite.Zero(value)
}

func (suite *SecretsExpanderTestSuite) assertExpansionSuccess(secretName string, expectedExpansionValue string) {
	value, failure := suite.prepareWorkingExpander()(secretName, suite.projectFile)
	suite.Equal(expectedExpansionValue, value)
	suite.Nil(failure)
}

func (suite *SecretsExpanderTestSuite) TestKeypairNotFound() {
	expanderFn := expander.NewVarExpander(suite.secretsClient)
	value, failure := expanderFn("undefined-secret", suite.projectFile)
	suite.Truef(failure.Type.Matches(keypairs.FailLoadNotFound), "unexpected failure type: %v", failure.Type)
	suite.Zero(value)
}

func (suite *SecretsExpanderTestSuite) TestSecretSpecNotDefinedInProject() {
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)
	// secret is in the database, but not defined in the project
	expanderFn := expander.NewVarExpander(suite.secretsClient)
	value, failure := expanderFn("foo", suite.projectFile)
	suite.True(failure.Type.Matches(expander.FailVarNotFound))
	suite.Zero(value)
}

func (suite *SecretsExpanderTestSuite) TestOrgNotFound() {
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg", 404)

	expanderFn := expander.NewVarExpander(suite.secretsClient)
	value, failure := expanderFn("undefined-secret", suite.projectFile)
	suite.True(failure.Type.Matches(api.FailOrganizationNotFound))
	suite.Zero(value)
}

func (suite *SecretsExpanderTestSuite) TestProjectNotFound() {
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg", 200)
	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010002/user_secrets", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg/projects/SecretProject", 404)

	expanderFn := expander.NewVarExpander(suite.secretsClient)
	value, failure := expanderFn("undefined-secret", suite.projectFile)
	suite.True(failure.Type.Matches(api.FailProjectNotFound))
	suite.Zero(value)
}

func (suite *SecretsExpanderTestSuite) TestDecodingFailed() {
	suite.assertExpansionFailure("bad-base64-encoded-secret", keypairs.FailKeyDecode)
}

func (suite *SecretsExpanderTestSuite) TestDecryptionFailed() {
	suite.assertExpansionFailure("invalid-encryption-secret", keypairs.FailDecrypt)
}

func (suite *SecretsExpanderTestSuite) TestSecretHasNoValue() {
	// secret is defined in the project, but not in the database
	suite.assertExpansionFailure("undefined-secret", secretsapi.FailUserSecretNotFound)
}

func (suite *SecretsExpanderTestSuite) TestOrgSecret() {
	suite.assertExpansionSuccess("org-secret", "amazing")
}

func (suite *SecretsExpanderTestSuite) TestProjectSecret() {
	// NOTE the user_secrets response has org and project scoped secrets with same name
	suite.assertExpansionSuccess("proj-secret", "proj-value")
}

func (suite *SecretsExpanderTestSuite) TestUserSecret() {
	// NOTE the user_secrets response has org, project, and user scoped secrets with same name
	suite.assertExpansionSuccess("user-secret", "user-value")
}

func (suite *SecretsExpanderTestSuite) TestUserProjectSecret() {
	// NOTE the user_secrets response has org, project, user, and user-project scoped secrets with same name
	suite.assertExpansionSuccess("user-proj-secret", "user-proj-value")
}

func (suite *SecretsExpanderTestSuite) TestOrgSecret_PrefersProjectScopeIfAvailable() {
	// NOTE the user_secrets response has org and project scoped secrets with same name
	suite.assertExpansionSuccess("org-secret-with-proj-value", "proj-value")
}

func (suite *SecretsExpanderTestSuite) TestProjSecret_PrefersUserScopeIfAvailable() {
	// NOTE the user_secrets response has project and user scoped secrets with same name
	suite.assertExpansionSuccess("proj-secret-with-user-value", "user-value")
}

func (suite *SecretsExpanderTestSuite) TestUserSecret_PrefersUserProjScopeIfAvailable() {
	// NOTE the user_secrets response has user and user-project scoped secrets with same name
	suite.assertExpansionSuccess("user-secret-with-user-proj-value", "user-proj-value")
}

func (suite *SecretsExpanderTestSuite) TestProjectSecret_FindsNoSecretIfOnlyOrgAvailable() {
	// NOTE the user_secrets response has user and user-project scoped secrets with same name
	suite.assertExpansionFailure("proj-secret-only-org-available", secretsapi.FailUserSecretNotFound)
}

func (suite *SecretsExpanderTestSuite) TestUserSecret_FindsNoSecretIfOnlyProjectAvailable() {
	// NOTE the user_secrets response has user and user-project scoped secrets with same name
	suite.assertExpansionFailure("user-secret-only-proj-available", secretsapi.FailUserSecretNotFound)
}

func (suite *SecretsExpanderTestSuite) TestUserProjSecret_AllowsUserIfUserProjectNotAvailable() {
	// NOTE the user_secrets response has user and user-project scoped secrets with same name
	suite.assertExpansionSuccess("user-proj-secret-only-user-available", "user-value")
}

func Test_SecretsExpander_TestSuite(t *testing.T) {
	suite.Run(t, new(SecretsExpanderTestSuite))
}
