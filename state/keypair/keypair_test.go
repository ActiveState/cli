package keypair_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/state/keypair"
	"github.com/stretchr/testify/suite"
)

type KeypairCommandTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
}

func (suite *KeypairCommandTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()

	secretsClient := secretsapi_test.InitializeTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	httpmock.Activate(secretsClient.BaseURI)
}

func (suite *KeypairCommandTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
}

func (suite *KeypairCommandTestSuite) TestExecute_NoArgs_AuthFailure() {
	cmd := keypair.Command

	httpmock.RegisterWithCode("GET", "/whoami", 401)

	var execErr error
	outStr, outErr := osutil.CaptureStderr(func() {
		cmd.GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Error(failures.Handled(), "failure occurred")

	suite.Contains(outStr, "You are not authenticated")
}

func (suite *KeypairCommandTestSuite) TestExecute_NoArgsDump_OutputsKeypair() {
	cmd := keypair.Command

	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithCode("GET", "/keypair", 200)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.NoError(failures.Handled(), "unexpected failure occurred")

	suite.Contains(outStr, "RSA PRIVATE KEY")
	suite.Contains(outStr, "ENCRYPTED")
	suite.Contains(outStr, "RSA PUBLIC KEY")
}

func (suite *KeypairCommandTestSuite) TestExecute_NoArgsDump_KeypairNotFound() {
	cmd := keypair.Command

	httpmock.RegisterWithCode("GET", "/whoami", 200)
	httpmock.RegisterWithCode("GET", "/keypair", 404)

	cmd.GetCobraCmd().SetArgs([]string{})
	execErr := cmd.Execute()
	suite.Require().NoError(execErr)
	suite.Require().Error(failures.Handled(), "expected failure")
	suite.Require().True(failures.IsFailure(failures.Handled()), "is a failure")
	failure := failures.Handled().(*failures.Failure)
	suite.True(failure.Type.Matches(secretsapi.FailNotFound), "should be a FailNotFound failure")
}

func Test_KeypairCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(KeypairCommandTestSuite))
}
