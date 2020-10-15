package secrets_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/locale"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/runners/secrets"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/suite"
)

type SecretsAccessTestSuite struct {
	suite.Suite

	projectFile *projectfile.Project

	secretsClient *secretsapi.Client
	platformMock  *httpmock.HTTPMock
	authMock      *authMock.Mock
}

func (suite *SecretsAccessTestSuite) BeforeTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")
	err = os.Chdir(filepath.Join(root, "internal", "runners", "secrets", "testdata", "access"))
	suite.Require().NoError(err, "Should chdir")

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())

	suite.authMock = authMock.Init()
	suite.authMock.MockLoggedin()
}

func (suite *SecretsAccessTestSuite) runCommand(expectedExitCode int, expectedOutput string) {
	cmd := secrets.NewCommand(suite.secretsClient, new(string))

	ex := exiter.New()
	cmd.Config().Exiter = ex.Exit

	out := capturer.CaptureOutput(func() {
		code := ex.WaitForExit(func() {
			suite.NoError(cmd.Config().Execute())
		})
		suite.Equal(expectedExitCode, code, fmt.Sprintf("Expects exit code %d", expectedExitCode))
	})

	suite.Contains(out, expectedOutput)
}

func (suite *SecretsAccessTestSuite) TestExecuteNoAccess() {
	suite.platformMock.RegisterWithCode("GET", "/organizations/AccessOrg", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/AccessOrg/members", 200)

	suite.runCommand(1, locale.T("secrets_warning_no_access"))
}

func (suite *SecretsAccessTestSuite) TestExecuteAccessError() {
	suite.platformMock.RegisterWithCode("GET", "/organizations/AccessOrg/members", 401)

	suite.runCommand(1, locale.T("secrets_err_access"))

	failure := failures.Handled().(*failures.Failure)
	suite.Equalf(api.FailAuth, failure.Type, "unexpected failure type: %v", failure.Type)
}

func TestSecretsAccessTestSuite(t *testing.T) {
	suite.Run(t, new(SecretsAccessTestSuite))
}
