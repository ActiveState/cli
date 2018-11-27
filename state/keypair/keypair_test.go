package keypair_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/state/keypair"
	"github.com/stretchr/testify/suite"
)

type KeypairCommandTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
}

func (suite *KeypairCommandTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()
	secretsClient := secretsapi_test.NewTestClient("http", constants.SecretsAPIHostTesting, constants.SecretsAPIPath, "bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	httpmock.Activate(secretsClient.BaseURI)
}

func (suite *KeypairCommandTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *KeypairCommandTestSuite) TestCommandConfig() {
	cmd := keypair.NewCommand(suite.secretsClient)
	conf := cmd.Config()
	suite.Equal("keypair", conf.Name)
	suite.Equal("keypair_cmd_description", conf.Description, "i18n symbol")

	ccCmds := conf.GetCobraCmd().Commands()
	suite.Require().Len(ccCmds, 1, "number of subcommands")
	suite.Equal("generate", ccCmds[0].Name())
	suite.False(ccCmds[0].HasFlags())

	suite.Len(conf.Flags, 0, "number of command flags supported")
	suite.Len(conf.Arguments, 0, "number of commands args supported")
}

func (suite *KeypairCommandTestSuite) TestExecute_NoArgs_AuthFailure() {
	cmd := keypair.NewCommand(suite.secretsClient)

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
}

func (suite *KeypairCommandTestSuite) TestExecute_NoArgsDump_OutputsKeypair() {
	cmd := keypair.NewCommand(suite.secretsClient)

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
}

func (suite *KeypairCommandTestSuite) TestExecute_NoArgsDump_KeypairNotFound() {
	cmd := keypair.NewCommand(suite.secretsClient)

	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithCode("GET", "/keypair", 404)

	cmd.Config().GetCobraCmd().SetArgs([]string{})
	execErr := cmd.Config().Execute()
	suite.Require().NoError(execErr)
	suite.Require().Error(failures.Handled(), "expected failure")
	suite.Require().True(failures.IsFailure(failures.Handled()), "is a failure")
	failure := failures.Handled().(*failures.Failure)
	suite.True(failure.Type.Matches(secretsapi.FailNotFound), "should be a FailNotFound failure")
}

func (suite *KeypairCommandTestSuite) TestExecute_Generate_SavesNewKeypair() {
	cmd := keypair.NewCommand(suite.secretsClient)

	var bodyKeypair *models.Keypair
	var bodyErr error
	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithResponder("PUT", "/keypair", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &bodyKeypair)
		return 204, "keypair"
	})

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"generate"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Require().NoError(bodyErr)
	suite.NoError(failures.Handled(), "unexpected failure occurred")

	suite.Contains(*bodyKeypair.EncryptedPrivateKey, "RSA PRIVATE KEY")
	suite.Contains(*bodyKeypair.PublicKey, "RSA PUBLIC KEY")
	suite.Contains(outStr, "Keypair generated successfully")
}

func (suite *KeypairCommandTestSuite) TestExecute_Generate_SaveFails() {
	cmd := keypair.NewCommand(suite.secretsClient)

	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithCode("PUT", "/keypair", 400)

	cmd.Config().GetCobraCmd().SetArgs([]string{"generate"})
	execErr := cmd.Config().Execute()
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "expected failure")
	suite.Require().True(failures.IsFailure(failures.Handled()), "is a failure")
	failure := failures.Handled().(*failures.Failure)
	suite.True(failure.Type.Matches(secretsapi.FailSave), "should be a FailSave failure")
}

func Test_KeypairCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairCommandTestSuite))
}
