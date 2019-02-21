package variables_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/state/variables"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"
)

type VarSetCommandTestSuite struct {
	suite.Suite

	testdataDir string
	configDir   string

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
}

func (suite *VarSetCommandTestSuite) SetupSuite() {
	path, err := environment.GetRootPath()
	suite.Require().NoError(err, "error obtaining root path")

	suite.testdataDir = filepath.Join(path, "state", "variables", "testdata")
	suite.configDir = filepath.Join(suite.testdataDir, "generated", "config")
}

func (suite *VarSetCommandTestSuite) BeforeTest(suiteName, testName string) {
	failures.ResetHandled()

	// support test projectfile access
	srcProjectFile := filepath.Join(suite.testdataDir, constants.ConfigFileName)
	dstProjectFile := filepath.Join(suite.configDir, constants.ConfigFileName)
	suite.Require().Nil(fileutils.CopyFile(srcProjectFile, dstProjectFile), "unexpected failure generating projectfile")
	suite.Require().NoError(os.Chdir(suite.configDir))

	// setup api clients and http mocks
	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServicePlatform).String())

	suite.platformMock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")
}

func (suite *VarSetCommandTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
}

func (suite *VarSetCommandTestSuite) TestCommandConfig() {
	cc := variables.NewCommand(suite.secretsClient).Config().GetCobraCmd().Commands()[1]

	suite.Equal("set", cc.Name())
	suite.Equal(locale.T("variables_set_cmd_description"), cc.Short, "translation")

	suite.Require().Len(cc.Commands(), 0, "number of subcommands")

	suite.Require().False(cc.HasAvailableFlags())
}

func (suite *VarSetCommandTestSuite) TestExecute_RequiresNameAndValue() {
	cmd := variables.NewCommand(suite.secretsClient)
	cmd.Config().GetCobraCmd().SetArgs([]string{"set"})
	err := cmd.Config().Execute()
	suite.EqualError(err, "Argument missing: variables_set_arg_name_name\nArgument missing: variables_set_arg_value_name\n")
	suite.NoError(failures.Handled(), "No failure occurred")
}

func (suite *VarSetCommandTestSuite) TestExecute_UndefinedVar() {
	cmd := variables.NewCommand(suite.secretsClient)
	cmd.Config().GetCobraCmd().SetArgs([]string{"set", "NEWVAR", "/new/path"})
	execErr := cmd.Config().Execute()
	suite.Require().NoError(execErr, "error executing command")
	suite.Require().Error(failures.Handled(), "expected error executing command")
	suite.Equal(failures.FailCmd, failures.Handled().(*failures.Failure).Type)
}

func (suite *VarSetCommandTestSuite) TestExecute_DefinedLocalVar_Success() {
	cmd := variables.NewCommand(suite.secretsClient)
	cmd.Config().GetCobraCmd().SetArgs([]string{"set", "PYTHONPATH", "/new/path"})
	execErr := cmd.Config().Execute()
	suite.Require().NoError(execErr, "error executing command")
	suite.Require().Nil(failures.Handled(), "unexpected failure executing command")

	projectfile.Reset()
	prj, failure := project.GetSafe()
	suite.Require().Nil(failure, "error loading project")

	pythonPathVar := prj.VariableByName("PYTHONPATH")
	pythonPath, failure := pythonPathVar.Value()
	suite.Require().Nil(failure, "error retrieving var")
	suite.Equal("/new/path", pythonPath)
}

func (suite *VarSetCommandTestSuite) TestExecute_ForSecret_FetchOrg_NotAuthenticated() {
	cmd := variables.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 401)

	var execErr error
	outStr, outErr := osutil.CaptureStderr(func() {
		cmd.Config().GetCobraCmd().SetArgs([]string{"set", "org-secret", "value1"})
		execErr = cmd.Config().Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)
	suite.Require().Error(failures.Handled(), "expected failure")

	suite.Contains(outStr, locale.T("err_api_not_authenticated"))
}

func (suite *VarSetCommandTestSuite) TestExecute_UpdateOrgSecret_Succeeds() {
	suite.assertSaveSucceeds("org-secret", false, false)
}

func (suite *VarSetCommandTestSuite) TestExecute_UpdateProjectSecret_Succeeds() {
	suite.assertSaveSucceeds("proj-secret", true, false)
}

func (suite *VarSetCommandTestSuite) TestExecute_UpdateUserSecret_Succeeds() {
	suite.assertSaveSucceeds("user-org-secret", false, true)
}

func (suite *VarSetCommandTestSuite) TestExecute_UpdateUserProjectSecret_Succeeds() {
	suite.assertSaveSucceeds("user-proj-secret", true, true)
}

func (suite *VarSetCommandTestSuite) assertSaveSucceeds(secretName string, isProject, isUserOnly bool) {
	cmd := variables.NewCommand(suite.secretsClient)

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	if isProject {
		suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/projects/CodeIntel", 200)
	}
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	var userChanges []*models.UserSecretChange
	var bodyErr error
	suite.secretsMock.RegisterWithResponder("PATCH", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &userChanges)
		return 204, "empty-response"
	})

	var sharedChanges []*models.UserSecretShare
	if !isUserOnly {
		// assert secrets get pushed for other users
		suite.secretsMock.RegisterWithCode("GET", "/whoami", 200)
		suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
		suite.secretsMock.RegisterWithCode("GET", "/publickeys/00020002-0002-0002-0002-000200020002", 200)
		suite.secretsMock.RegisterWithResponder("PATCH", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets/00020002-0002-0002-0002-000200020002", func(req *http.Request) (int, string) {
			reqBody, _ := ioutil.ReadAll(req.Body)
			json.Unmarshal(reqBody, &sharedChanges)
			return 204, "empty-response"
		})
	}

	cmd.Config().GetCobraCmd().SetArgs([]string{"set", secretName, "secret-value"})
	execErr := cmd.Config().Execute()

	suite.Require().NoError(execErr)
	suite.Require().NoError(bodyErr)
	suite.NoError(failures.Handled())

	suite.Require().Len(userChanges, 1)
	suite.NotZero(*userChanges[0].Value)
	suite.Equal(secretName, *userChanges[0].Name)
	suite.Equal(isUserOnly, *userChanges[0].IsUser)
	if isProject {
		suite.Equal(strfmt.UUID("00020002-0002-0002-0002-000200020002"), userChanges[0].ProjectID)
	} else {
		suite.Zero(userChanges[0].ProjectID)
	}

	if !isUserOnly {
		suite.Require().Len(sharedChanges, 1)
		suite.NotZero(*sharedChanges[0].Value)
		suite.Equal(secretName, *sharedChanges[0].Name)
		suite.Equal(userChanges[0].ProjectID, sharedChanges[0].ProjectID)
	} else {
		suite.Require().Len(sharedChanges, 0)
	}

}

func Test_VarSetCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(VarSetCommandTestSuite))
}
