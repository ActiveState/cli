package keypair_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	promptMock "github.com/ActiveState/cli/internal/prompt/mock"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/state/keypair"
	keyp "github.com/ActiveState/cli/state/keypair"
	"github.com/stretchr/testify/suite"
)

var pmock = promptMock.Init()

type KeypairAuthTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
}

func (suite *KeypairAuthTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()

	secretsClient := secretsapi_test.InitializeTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient
	keyp.Prompter = pmock
	httpmock.Activate(secretsClient.BaseURI)
}

func (suite *KeypairAuthTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *KeypairAuthTestSuite) TestExecute_APIAuthFailure() {
	cmd := keypair.Command

	httpmock.RegisterWithCode("GET", "/whoami", 401)

	cmd.GetCobraCmd().SetArgs([]string{"auth"})
	execErr := cmd.Execute()

	suite.Require().Error(execErr, "expected failure")
	suite.Require().True(failures.IsFailure(failures.Handled()), "is a failure")

	failure := failures.Handled().(*failures.Failure)
	suite.Truef(failure.Type.Matches(api.FailAuth), "unexpected failure type: %v", failure)
}

func (suite *KeypairAuthTestSuite) TestExecute_NoKeypairFound() {
	cmd := keypair.Command

	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithCode("GET", "/keypair", 404)

	cmd.GetCobraCmd().SetArgs([]string{"auth"})
	execErr := cmd.Execute()

	suite.Require().Error(execErr, "expected failure")
	suite.Require().True(failures.IsFailure(failures.Handled()), "is a failure")

	failure := failures.Handled().(*failures.Failure)
	suite.Truef(failure.Type.Matches(secretsapi.FailNotFound), "unexpected failure type: %v", failure)
}

func (suite *KeypairAuthTestSuite) TestExecute_InvalidPassphrase() {
	cmd := keypair.Command

	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithCode("GET", "/keypair", 200)

	cmd.GetCobraCmd().SetArgs([]string{"auth"})
	var execErr error
	pmock.OnMethod("InputPassword").Once().Return("badpass", nil)
	execErr = cmd.Execute()

	suite.Require().Error(execErr, "expected failure")
	suite.Require().True(failures.IsFailure(failures.Handled()), "is a failure")

	failure := failures.Handled().(*failures.Failure)
	suite.Truef(failure.Type.Matches(keypairs.FailKeypairPassphrase), "unexpected failure type: %v", failure)
}

func (suite *KeypairAuthTestSuite) TestExecute_Success() {
	cmd := keypair.Command

	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithCode("GET", "/keypair", 200)

	cmd.GetCobraCmd().SetArgs([]string{"auth"})
	var execErr error
	pmock.OnMethod("InputPassword").Once().Return("foo", nil)
	execErr = cmd.Execute()
	suite.Require().NoError(execErr)

	suite.Require().NoError(failures.Handled())

	keyContents, fileErr := osutil.ReadConfigFile("private.key")
	suite.Require().NoError(fileErr)
	suite.Contains(keyContents, "RSA PRIVATE KEY")
	suite.NotContains(keyContents, "ENCRYPTED")
	suite.NotContains(keyContents, "RSA PUBLIC KEY")
}

func Test_KeypairAuthTestSuite(t *testing.T) {
	suite.Run(t, new(KeypairAuthTestSuite))
}
