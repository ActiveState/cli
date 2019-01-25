package variables_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/cli/state/variables"
	"github.com/stretchr/testify/suite"
)

type SecretsGetCommandTestSuite struct {
	suite.Suite

	projectFile *projectfile.Project

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
}

func (suite *SecretsGetCommandTestSuite) BeforeTest(suiteName, testName string) {
	locale.Set("en-US")
	failures.ResetHandled()

	projectFile, err := loadSecretsProject()
	suite.Require().Nil(err, "Unmarshalled project YAML")
	projectFile.Persist()
	suite.projectFile = projectFile

	// support test projectfile access
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.Prefix)
}

func (suite *SecretsGetCommandTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	projectfile.Reset()
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
}

func (suite *SecretsGetCommandTestSuite) prepareWorkingExpander() {
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg/projects/SecretProject", 200)

	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010002/user_secrets", 200)
}

func (suite *SecretsGetCommandTestSuite) assertExpansionFailure(secretName string, expectedFailureType *failures.FailureType) {
	suite.prepareWorkingExpander()

	cmd := variables.NewCommand(suite.secretsClient)
	cmd.Config().GetCobraCmd().SetArgs([]string{"get", secretName})
	execErr := cmd.Config().Execute()
	suite.Require().NoError(execErr)
	suite.Require().Error(failures.Handled(), "expected a failure")

	failure := failures.Handled().(*failures.Failure)
	suite.Truef(failure.Type.Matches(expectedFailureType), "unexpected failure type: %v", failure.Type)
}

func (suite *SecretsGetCommandTestSuite) assertExpansionSuccess(secretName string, expectedExpansionValue string) {
	suite.prepareWorkingExpander()
	cmd := variables.NewCommand(suite.secretsClient)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"get", secretName})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(execErr)
	suite.Require().NoError(outErr)
	suite.Require().NoError(failures.Handled(), "unexpected failure")

	suite.Equal(expectedExpansionValue, outStr)
}

func (suite *SecretsGetCommandTestSuite) TestCommandConfig() {
	cc := variables.NewCommand(suite.secretsClient).Config().GetCobraCmd().Commands()[0]

	suite.Equal("get", cc.Name())
	suite.Require().Len(cc.Commands(), 0, "number of subcommands")
	suite.Require().False(cc.HasAvailableFlags())
}

func (suite *SecretsSetCommandTestSuite) TestExecute_RequiresSecretName() {
	cmd := variables.NewCommand(suite.secretsClient)
	cmd.Config().GetCobraCmd().SetArgs([]string{"get"})
	err := cmd.Config().Execute()
	suite.EqualError(err, "Argument missing: secrets_get_arg_name_name\n")
	suite.NoError(failures.Handled(), "No failure occurred")
}

func (suite *SecretsGetCommandTestSuite) TestDecodingFailed() {
	suite.assertExpansionFailure("bad-base64-encoded-secret", keypairs.FailKeyDecode)
}

func (suite *SecretsGetCommandTestSuite) TestDecryptionFailed() {
	suite.assertExpansionFailure("invalid-encryption-secret", keypairs.FailDecrypt)
}

func (suite *SecretsGetCommandTestSuite) TestSecretHasNoValue() {
	// secret is defined in the project, but not in the database
	suite.assertExpansionFailure("undefined-secret", secretsapi.FailUserSecretNotFound)
}

func (suite *SecretsGetCommandTestSuite) TestOrgSecret() {
	suite.assertExpansionSuccess("org-secret", "amazing")
}

func (suite *SecretsGetCommandTestSuite) TestProjectSecret() {
	// NOTE the user_secrets response has org and project scoped secrets with same name
	suite.assertExpansionSuccess("proj-secret", "proj-value")
}

func (suite *SecretsGetCommandTestSuite) TestUserSecret() {
	// NOTE the user_secrets response has org, project, and user scoped secrets with same name
	suite.assertExpansionSuccess("user-secret", "user-value")
}

func (suite *SecretsGetCommandTestSuite) TestUserProjectSecret() {
	// NOTE the user_secrets response has org, project, user, and user-project scoped secrets with same name
	suite.assertExpansionSuccess("user-proj-secret", "user-proj-value")
}

func (suite *SecretsGetCommandTestSuite) TestOrgSecret_PrefersProjectScopeIfAvailable() {
	// NOTE the user_secrets response has org and project scoped secrets with same name
	suite.assertExpansionSuccess("org-secret-with-proj-value", "proj-value")
}

func (suite *SecretsGetCommandTestSuite) TestProjSecret_PrefersUserScopeIfAvailable() {
	// NOTE the user_secrets response has project and user scoped secrets with same name
	suite.assertExpansionSuccess("proj-secret-with-user-value", "user-value")
}

func (suite *SecretsGetCommandTestSuite) TestUserSecret_PrefersUserProjScopeIfAvailable() {
	// NOTE the user_secrets response has user and user-project scoped secrets with same name
	suite.assertExpansionSuccess("user-secret-with-user-proj-value", "user-proj-value")
}

func (suite *SecretsGetCommandTestSuite) TestProjectSecret_FindsNoSecretIfOnlyOrgAvailable() {
	// NOTE the user_secrets response has user and user-project scoped secrets with same name
	suite.assertExpansionFailure("proj-secret-only-org-available", secretsapi.FailUserSecretNotFound)
}

func (suite *SecretsGetCommandTestSuite) TestUserSecret_FindsNoSecretIfOnlyProjectAvailable() {
	// NOTE the user_secrets response has user and user-project scoped secrets with same name
	suite.assertExpansionFailure("user-secret-only-proj-available", secretsapi.FailUserSecretNotFound)
}

func (suite *SecretsGetCommandTestSuite) TestUserProjSecret_AllowsUserIfUserProjectNotAvailable() {
	// NOTE the user_secrets response has user and user-project scoped secrets with same name
	suite.assertExpansionSuccess("user-proj-secret-only-user-available", "user-value")
}
func Test_SecretsGetCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(SecretsGetCommandTestSuite))
}
