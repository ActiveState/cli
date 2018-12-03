package secrets_test

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/internal/variables"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/stretchr/testify/suite"
	yaml "gopkg.in/yaml.v2"
)

func loadSecretsProject() (*projectfile.Project, error) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
name: SecretProject
owner: SecretOrg
secrets:
  - name: undefined-secret
  - name: org-secret
  - name: proj-secret
    project: true
  - name: user-secret
    user: true
  - name: user-proj-secret
    project: true
    user: true
  - name: org-secret-with-proj-value
  - name: proj-secret-with-user-value
    project: true
  - name: user-secret-with-user-proj-value
    user: true
  - name: proj-secret-only-org-available
    project: true
  - name: user-secret-only-proj-available
    user: true
  - name: user-proj-secret-only-user-available
    user: true
    project: true
  - name: bad-base64-encoded-secret
  - name: invalid-encryption-secret
commands:
  - name: echo-org-secret
    value: echo ${secrets.org-secret}
  - name: echo-upper-org-secret
    value: echo ${secrets.ORG-SECRET}
`)

	return project, yaml.Unmarshal([]byte(contents), project)
}

type SecretsExpanderTestSuite struct {
	suite.Suite

	projectFile *projectfile.Project

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
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
	suite.platformMock = httpmock.Activate(api.Prefix)
}

func (suite *SecretsExpanderTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	projectfile.Reset()
	osutil.RemoveConfigFile("private.key")
}

func (suite *SecretsExpanderTestSuite) prepareWorkingExpander() variables.ExpanderFunc {
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg/projects/SecretProject", 200)

	osutil.CopyTestFileToConfigDir("self-private.key", "private.key", 0600)

	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010002/user_secrets", 200)
	return secrets.NewExpander(suite.secretsClient)
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
	expanderFn := secrets.NewExpander(suite.secretsClient)
	value, failure := expanderFn("undefined-secret", suite.projectFile)
	suite.Truef(failure.Type.Matches(keypairs.FailLoadNotFound), "unexpected failure type: %v", failure.Type)
	suite.Zero(value)
}

func (suite *SecretsExpanderTestSuite) TestSecretSpecNotDefinedInProject() {
	osutil.CopyTestFileToConfigDir("self-private.key", "private.key", 0600)
	// secret is in the database, but not defined in the project
	expanderFn := secrets.NewExpander(suite.secretsClient)
	value, failure := expanderFn("foo", suite.projectFile)
	suite.True(failure.Type.Matches(secrets.FailUnrecognizedSecretSpec))
	suite.Zero(value)
}

func (suite *SecretsExpanderTestSuite) TestOrgNotFound() {
	osutil.CopyTestFileToConfigDir("self-private.key", "private.key", 0600)
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg", 404)

	expanderFn := secrets.NewExpander(suite.secretsClient)
	value, failure := expanderFn("undefined-secret", suite.projectFile)
	suite.True(failure.Type.Matches(api.FailOrganizationNotFound))
	suite.Zero(value)
}

func (suite *SecretsExpanderTestSuite) TestProjectNotFound() {
	osutil.CopyTestFileToConfigDir("self-private.key", "private.key", 0600)
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg/projects/SecretProject", 404)

	expanderFn := secrets.NewExpander(suite.secretsClient)
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
