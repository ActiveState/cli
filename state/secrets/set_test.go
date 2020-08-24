package secrets_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/state/secrets"
)

type VarSetCommandTestSuite struct {
	suite.Suite

	testdataDir string
	configDir   string

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
	graphMock     *graphMock.Mock
}

func (suite *VarSetCommandTestSuite) SetupSuite() {
	path, err := environment.GetRootPath()
	suite.Require().NoError(err, "error obtaining root path")

	suite.testdataDir = filepath.Join(path, "state", "secrets", "testdata")
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
	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())

	suite.platformMock.Register("POST", "/login")
	authentication.Get().AuthenticateWithToken("")

	suite.graphMock = graphMock.Init()
	suite.graphMock.ProjectByOrgAndName(graphMock.NoOptions)
}

func (suite *VarSetCommandTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
	suite.graphMock.Close()
}

func (suite *VarSetCommandTestSuite) TestCommandConfig() {
	cc := secrets.NewCommand(suite.secretsClient, new(string)).Config().GetCobraCmd().Commands()[1]

	suite.Equal("set", cc.Name())
	suite.Equal(locale.T("secrets_set_cmd_description"), cc.Short, "translation")

	suite.Require().Len(cc.Commands(), 0, "number of subcommands")

	suite.Require().False(cc.HasAvailableFlags())
}

func (suite *VarSetCommandTestSuite) TestExecute_SetSecret() {
	cmd := secrets.NewCommand(suite.secretsClient, new(string))

	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState", 200)
	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	var userChanges []*secrets_models.UserSecretChange
	var bodyErr error
	suite.secretsMock.RegisterWithResponder("PATCH", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &userChanges)
		return 204, "empty-response"
	})

	var sharedChanges []*secrets_models.UserSecretShare

	// assert secrets get pushed for other users
	suite.secretsMock.RegisterWithCode("GET", "/whoami", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/ActiveState/members", 200)
	suite.secretsMock.RegisterWithCode("GET", "/publickeys/00020002-0002-0002-0002-000200020002", 200)
	suite.secretsMock.RegisterWithResponder("PATCH", "/organizations/00010001-0001-0001-0001-000100010001/user_secrets/00020002-0002-0002-0002-000200020002", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		json.Unmarshal(reqBody, &sharedChanges)
		return 204, "empty-response"
	})

	cmd.Config().GetCobraCmd().SetArgs([]string{"set", "project.secret-name", "secret-value"})
	execErr := cmd.Config().Execute()

	suite.Require().NoError(execErr)
	suite.Require().NoError(bodyErr)
	suite.NoError(failures.Handled())

	suite.Require().Len(userChanges, 1)
	suite.NotZero(*userChanges[0].Value)
	suite.Equal("secret-name", *userChanges[0].Name)
	suite.Equal(false, *userChanges[0].IsUser)
	suite.Equal(strfmt.UUID("00010001-0001-0001-0001-000100010001"), userChanges[0].ProjectID)

	suite.Require().Len(sharedChanges, 1)
	suite.NotZero(*sharedChanges[0].Value)
	suite.Equal("secret-name", *sharedChanges[0].Name)
	suite.Equal(userChanges[0].ProjectID, sharedChanges[0].ProjectID)
}

func Test_VarSetCommand_TestSuite(t *testing.T) {
	suite.Run(t, new(VarSetCommandTestSuite))
}
