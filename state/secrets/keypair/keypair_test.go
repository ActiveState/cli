package keypair_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/state/secrets/keypair"
	"github.com/stretchr/testify/suite"
)

type KeypairCommandTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
}

func (suite *KeypairCommandTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()
	secretsClient := secretsapi.NewTestClient("http", constants.SecretsAPIHostTesting, constants.SecretsAPIPath, "bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	httpmock.Activate(secretsClient.BaseURI)
}

func (suite *KeypairCommandTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *KeypairCommandTestSuite) TestCommandConfig() {
	cmd, err := keypair.NewRSACommand(suite.secretsClient)
	suite.Require().NoError(err)

	conf := cmd.Config()
	suite.Equal("keypair", conf.Name)
	suite.Equal("secrets_keypair_cmd_description", conf.Description, "i18n symbol")

	suite.Len(conf.GetCobraCmd().Commands(), 0, "number of subcommands")

	suite.Require().Len(conf.Flags, 1, "number of command flags supported")
	suite.Equal("generate", conf.Flags[0].Name)
	suite.Equal("secrets_keypair_generate_flag_usage", conf.Flags[0].Description, "i18n symbol")
	suite.Equal(commands.TypeBool, conf.Flags[0].Type)

	suite.Len(conf.Arguments, 0, "number of commands args supported")
}

func (suite *KeypairCommandTestSuite) TestExecute_NoArgs_AuthFailure() {
	mockKeypair := new(MockKeypair) // expect nothing
	cmd := keypair.NewCommand(suite.secretsClient, NewMockGeneratorFunc(mockKeypair))

	httpmock.RegisterWithCode("GET", "/whoami", 401)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "failure occurred")

	suite.Contains(outStr, "You are not authenticated")
	suite.True(mockKeypair.AssertExpectations(suite.T()), "mock keypair expectations")
}

func (suite *KeypairCommandTestSuite) TestExecute_NoArgsDump_OutputsKeypair() {
	mockKeypair := new(MockKeypair) // expect nothing
	cmd := keypair.NewCommand(suite.secretsClient, NewMockGeneratorFunc(mockKeypair))

	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithCode("GET", "/keypair", 200)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.NoError(failures.Handled(), "unexpected failure occurred")

	suite.Contains(outStr, "RSA PRIVATE KEY")
	suite.Contains(outStr, "RSA PUBLIC KEY")
	suite.True(mockKeypair.AssertExpectations(suite.T()), "mock keypair expectations")
}

func (suite *KeypairCommandTestSuite) TestExecute_NoArgsDump_KeypairNotFound() {
	mockKeypair := new(MockKeypair) // expect nothing
	cmd := keypair.NewCommand(suite.secretsClient, NewMockGeneratorFunc(mockKeypair))

	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithCode("GET", "/keypair", 404)

	cmd.Config().GetCobraCmd().SetArgs([]string{})
	execErr := cmd.Config().Execute()
	suite.Require().NoError(execErr)
	suite.Require().Error(failures.Handled(), "expected failure")
	suite.Require().True(failures.IsFailure(failures.Handled()), "is a failure")
	failure := failures.Handled().(*failures.Failure)
	suite.True(failure.Type.Matches(secretsapi.FailNotFound), "should be a FailNotFound failure")

	suite.True(mockKeypair.AssertExpectations(suite.T()), "mock keypair expectations")
}

func (suite *KeypairCommandTestSuite) TestExecute_Generate_SavesNewKeypair() {
	mockKeypair := new(MockKeypair)
	cmd := keypair.NewCommand(suite.secretsClient, NewMockGeneratorFunc(mockKeypair))

	mockKeypair.On("EncodePrivateKey").Return("-- MOCK PRIVATE KEY --")
	mockKeypair.On("EncodePublicKey").Return("-- MOCK PUBLIC KEY --", error(nil))

	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithCode("PUT", "/keypair", 204)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"--generate"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.NoError(failures.Handled(), "unexpected failure occurred")

	suite.Contains(outStr, "Keypair generated successfully")
	suite.True(mockKeypair.AssertExpectations(suite.T()), "mock keypair expectations")
}

func (suite *KeypairCommandTestSuite) TestExecute_Generate_SaveFails() {
	mockKeypair := new(MockKeypair)
	cmd := keypair.NewCommand(suite.secretsClient, NewMockGeneratorFunc(mockKeypair))

	mockKeypair.On("EncodePrivateKey").Return("-- MOCK PRIVATE KEY --")
	mockKeypair.On("EncodePublicKey").Return("-- MOCK PUBLIC KEY --", error(nil))

	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithCode("PUT", "/keypair", 400)

	cmd.Config().GetCobraCmd().SetArgs([]string{"--generate"})
	execErr := cmd.Config().Execute()
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "expected failure")
	suite.Require().True(failures.IsFailure(failures.Handled()), "is a failure")
	failure := failures.Handled().(*failures.Failure)
	suite.True(failure.Type.Matches(secretsapi.FailSave), "should be a FailSave failure")

	suite.True(mockKeypair.AssertExpectations(suite.T()), "mock keypair expectations")
}

func Test_KeypairCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairCommandTestSuite))
}
