package secrets_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/secrets"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type SecretsGetCommandTestSuite struct {
	suite.Suite

	projectFile *projectfile.Project

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
	graphMock     *graphMock.Mock
}

func (suite *SecretsGetCommandTestSuite) BeforeTest(suiteName, testName string) {
	locale.Set("en-US")
	failures.ResetHandled()

	projectFile, err := loadSecretsProject()
	suite.Require().Nil(err, "unmarshalling custom project yaml")
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
	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())

	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg/members", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)

	suite.platformMock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	suite.graphMock = graphMock.Init()
	suite.graphMock.ProjectByOrgAndName(graphMock.NoOptions)
}

func (suite *SecretsGetCommandTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	projectfile.Reset()
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
	suite.graphMock.Close()
}

func (suite *SecretsGetCommandTestSuite) prepareWorkingExpander() {
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg", 200)
	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010002/user_secrets", 200)
}

func (suite *SecretsGetCommandTestSuite) assertExpansionFailure(secretName string, expectedFailureType *failures.FailureType, expectedExitCode int) {
	suite.prepareWorkingExpander()

	cmd := secrets.NewCommand(suite.secretsClient, new(string))
	cmd.Config().GetCobraCmd().SetArgs([]string{"get", secretName})

	ex := exiter.New()
	cmd.Config().Exiter = ex.Exit

	exitCode := ex.WaitForExit(func() {
		cmd.Config().Execute()
	})
	suite.Equal(expectedExitCode, exitCode, "expected exit code to match")

	handled := failures.Handled()
	failure, ok := handled.(*failures.Failure)
	suite.Require().Truef(ok, "got %v, wanted failure", handled)
	suite.True(failures.Matches(failure, expectedFailureType), "unexpected failure type: %v", failure.Type)
}

func (suite *SecretsGetCommandTestSuite) assertExpansion(secretName string, expectedExpansionValue string, expectedExitCode int) {
	suite.prepareWorkingExpander()
	cmd := secrets.NewCommand(suite.secretsClient, new(string))

	var exitCode int
	ex := exiter.New()
	cmd.Config().Exiter = ex.Exit

	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"get", secretName})
		exitCode = ex.WaitForExit(func() {
			cmd.Config().Execute()
		})
	})
	suite.Equal(expectedExitCode, exitCode, "expected exit code to match")
	suite.Require().NoError(outErr)
	suite.Require().NoError(failures.Handled(), "unexpected failure")

	suite.Equal(expectedExpansionValue, strings.TrimSpace(outStr))
}

func (suite *SecretsGetCommandTestSuite) TestCommandConfig() {
	cc := secrets.NewCommand(suite.secretsClient, new(string)).Config().GetCobraCmd().Commands()[0]

	suite.Equal("get", cc.Name())
	suite.Require().Len(cc.Commands(), 0, "number of subcommands")
}

func (suite *SecretsGetCommandTestSuite) TestDecodingFailed() {
	suite.assertExpansionFailure("project.bad-base64-encoded-secret", keypairs.FailKeyDecode, 1)
}

func (suite *SecretsGetCommandTestSuite) TestDecryptionFailed() {
	suite.assertExpansionFailure("project.invalid-encryption-secret", keypairs.FailDecrypt, 1)
}

func (suite *SecretsGetCommandTestSuite) TestSecretHasNoValue() {
	// secret is not defined (has no value)
	suite.assertExpansion("user.undefined-secret", "", 1)
}

func (suite *SecretsGetCommandTestSuite) TestSecretWithValue() {
	// NOTE the user_secrets response has org and project scoped secrets with same name
	suite.assertExpansion("project.secret-name", "proj-value", -1)
}

func Test_SecretsGetCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(SecretsGetCommandTestSuite))
}
