package secrets_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/runners/secrets"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type VariablesCommandTestSuite struct {
	suite.Suite

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
	authMock      *authMock.Mock
	graphMock     *graphMock.Mock
}

func (suite *VariablesCommandTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()
	projectfile.Reset()

	err := osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)
	suite.Require().NoError(err, "issue creating local private key")

	// support test projectfile access
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should detect root path")
	err = os.Chdir(filepath.Join(root, "internal", "runners", "secrets", "testdata"))
	suite.Require().NoError(err, "Should chdir")

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())

	suite.authMock = authMock.Init()
	suite.authMock.MockLoggedin()

	suite.graphMock = graphMock.Init()
	suite.graphMock.ProjectByOrgAndName(graphMock.NoOptions)
}

func (suite *VariablesCommandTestSuite) AfterTest(suiteName, testName string) {
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
	httpmock.DeActivate()
	suite.authMock.Close()
	suite.graphMock.Close()
}

func (suite *VariablesCommandTestSuite) TestExecute_ListAll() {
	cmd := secrets.NewCommand(suite.secretsClient, new(string))

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
	suite.secretsMock.RegisterWithResponder("GET", "/definitions/00010001-0001-0001-0001-000100010001", func(req *http.Request) (int, string) {
		return 200, "definitions/00010001-0001-0001-0001-000100010001"
	})
	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", 200)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Require().Nil(failures.Handled(), "unexpected failure occurred")

	suite.Contains(strings.TrimSpace(outStr), "proj-secret")
	suite.Contains(strings.TrimSpace(outStr), "proj-secret-description")
	suite.Contains(strings.TrimSpace(outStr), "user-secret")
	suite.Contains(strings.TrimSpace(outStr), "user-secret-description")
}

func (suite *VariablesCommandTestSuite) TestExecute_ListFilter() {
	cmd := secrets.NewCommand(suite.secretsClient, new(string))

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
	suite.secretsMock.RegisterWithResponder("GET", "/definitions/00010001-0001-0001-0001-000100010001", func(req *http.Request) (int, string) {
		return 200, "definitions/00010001-0001-0001-0001-000100010001"
	})
	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", 200)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"--filter-usedby", "scripts.secret-indirect"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Require().Nil(failures.Handled(), "unexpected failure occurred")

	suite.Contains(strings.TrimSpace(outStr), "proj-secret")
	suite.Contains(strings.TrimSpace(outStr), "proj-secret-description")
	suite.Contains(strings.TrimSpace(outStr), "Defined")
	suite.NotContains(strings.TrimSpace(outStr), "user-secret")
}

func (suite *VariablesCommandTestSuite) TestExecute_ListAllJSON() {
	cmd := secrets.NewCommand(suite.secretsClient, new(string))

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
	suite.secretsMock.RegisterWithResponder("GET", "/definitions/00010001-0001-0001-0001-000100010001", func(req *http.Request) (int, string) {
		return 200, "definitions/00010001-0001-0001-0001-000100010001"
	})
	suite.secretsMock.RegisterWithCode("GET", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", 200)

	var execErr error
	outStr, outErr := osutil.CaptureStdout(func() {
		output := "json"
		cmd.Flags.Output = &output
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Require().Nil(failures.Handled(), "unexpected failure occurred")

	secretsJson := []secrets.SecretExport{}
	err := json.Unmarshal([]byte(outStr), &secretsJson)
	suite.Require().NoError(err)
	suite.Len(secretsJson, 2)
}

func Test_VariablesCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(VariablesCommandTestSuite))
}
