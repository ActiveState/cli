package project_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type ValueOrNilTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
}

func (suite *ValueOrNilTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()

	projectFile, err := suite.loadSecretsProject()
	suite.Require().Nil(err, "unmarshalling custom project yaml")
	projectFile.Persist()

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient
	secretsClient.Persist()

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())

	suite.platformMock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")
}

func (suite *ValueOrNilTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	projectfile.Reset()
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
}

func (suite *ValueOrNilTestSuite) prepareAPI() {
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg/projects/SecretProject", 200)
	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010002/user_secrets", 200)
}

func (suite *ValueOrNilTestSuite) expandVariable(secretName string) (*string, *failures.Failure) {
	suite.prepareAPI()
	prj, failure := project.GetSafe()
	suite.Require().Nil(failure, "failure getting project safely")
	variable := prj.VariableByName(secretName)
	suite.Require().NotNil(variable, "expected a variable")
	return variable.ValueOrNil()
}

func (suite *ValueOrNilTestSuite) assertExpansionFailure(secretName string, expectedFailureType *failures.FailureType) {
	value, failure := suite.expandVariable(secretName)
	suite.Require().Nil(value, "expected value to be nil")
	suite.Require().NotNil(failure, "expected a failure")
	suite.Equalf(expectedFailureType, failure.Type, "unexpected failure type: %v", failure.Type)
}

func (suite *ValueOrNilTestSuite) assertExpansion(secretName string, expectedExpansionValue string) {
	value, failure := suite.expandVariable(secretName)
	suite.Require().Nil(failure, "unexpected failure expanding variable value")
	suite.Require().NotNil(value, "value should not be nil")
	suite.Equal(expectedExpansionValue, *value)
}

func (suite *ValueOrNilTestSuite) assertNilExpansion(secretName string) {
	value, failure := suite.expandVariable(secretName)
	suite.Require().Nil(failure, "unexpected failure expanding variable value")
	suite.Nil(value, "value should be nil")
}

func (suite *ValueOrNilTestSuite) TestDecodingFailed() {
	suite.assertExpansionFailure("bad-base64-encoded-secret", keypairs.FailKeyDecode)
}

func (suite *ValueOrNilTestSuite) TestDecryptionFailed() {
	suite.assertExpansionFailure("invalid-encryption-secret", keypairs.FailDecrypt)
}

func (suite *ValueOrNilTestSuite) TestSecretHasNoValue() {
	// secret is defined in the project, but not in the database
	suite.assertNilExpansion("undefined-secret")
}

func (suite *ValueOrNilTestSuite) TestOrgSecret() {
	suite.assertExpansion("org-secret", "amazing")
}

func (suite *ValueOrNilTestSuite) TestProjectSecret() {
	// NOTE the user_secrets response has org and project scoped secrets with same name
	suite.assertExpansion("proj-secret", "proj-value")
}

func (suite *ValueOrNilTestSuite) TestUserSecret() {
	// NOTE the user_secrets response has org, project, and user scoped secrets with same name
	suite.assertExpansion("user-secret", "user-value")
}

func (suite *ValueOrNilTestSuite) TestUserProjectSecret() {
	// NOTE the user_secrets response has org, project, user, and user-project scoped secrets with same name
	suite.assertExpansion("user-proj-secret", "user-proj-value")
}

func (suite *ValueOrNilTestSuite) TestProjectSecret_FindsNoSecretIfOnlyOrgAvailable() {
	// NOTE the user_secrets response has user and user-project scoped secrets with same name
	suite.assertNilExpansion("proj-secret-only-org-available")
}

func (suite *ValueOrNilTestSuite) TestUserSecret_FindsNoSecretIfOnlyProjectAvailable() {
	// NOTE the user_secrets response has project scoped secret with same name
	suite.assertNilExpansion("user-secret-only-proj-available")
}

func (suite *ValueOrNilTestSuite) loadSecretsProject() (*projectfile.Project, error) {
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
      share: organization
  - name: user-secret-with-user-proj-value
    value:
      pullfrom: organization
  - name: proj-secret-only-org-available
    value:
      pullfrom: project
      share: organization
  - name: user-secret-only-proj-available
    value:
      pullfrom: organization
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
scripts:
  - name: echo-org-secret
    value: echo ${secrets.org-secret}
  - name: echo-upper-org-secret
    value: echo ${secrets.ORG-SECRET}
`)

	err := yaml.Unmarshal([]byte(contents), project)
	if err != nil {
		return nil, err
	}

	return project, project.Parse()
}

func Test_ValueOrNilTestSuite(t *testing.T) {
	suite.Run(t, new(ValueOrNilTestSuite))
}
