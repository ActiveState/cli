package project_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	promptMock "github.com/ActiveState/cli/internal/prompt/mock"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	secretsModels "github.com/ActiveState/cli/pkg/platform/api/secrets/secrets_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type VarPromptingExpanderTestSuite struct {
	suite.Suite

	projectFile   *projectfile.Project
	project       *project.Project
	promptMock    *promptMock.Mock
	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
	graphMock     *mock.Mock
}

func (suite *VarPromptingExpanderTestSuite) BeforeTest(suiteName, testName string) {
	locale.Set("en-US")
	failures.ResetHandled()

	suite.promptMock = promptMock.Init()
	pjFile, err := loadSecretsProject()
	suite.Require().Nil(err, "Unmarshalled project YAML")
	pjFile.Persist()
	suite.projectFile = pjFile
	var fail *failures.Failure
	suite.project, fail = project.New(pjFile, outputhelper.NewCatcher(), suite.promptMock)
	suite.NoError(fail.ToError(), "no failure should occur when loading project")

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())

	suite.platformMock.Register("POST", "/login")
	suite.platformMock.Register("GET", "/organizations/SecretOrg/members")
	authentication.Get().AuthenticateWithToken("")

	suite.graphMock = mock.Init()
	suite.graphMock.ProjectByOrgAndName(mock.NoOptions)
}

func (suite *VarPromptingExpanderTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	projectfile.Reset()
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
	suite.graphMock.Close()
}

func (suite *VarPromptingExpanderTestSuite) prepareWorkingExpander() project.ExpanderFunc {
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg", 200)

	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	suite.secretsMock.RegisterWithResponder("GET", "/organizations/00010001-0001-0001-0001-000100010002/user_secrets", func(req *http.Request) (int, string) {
		return 200, "user_secrets-empty"
	})
	return project.NewSecretPromptingExpander(suite.secretsClient)
}

func (suite *VarPromptingExpanderTestSuite) assertExpansionSaveFailure(secretName, expectedValue string) {
	suite.secretsMock.RegisterWithResponder("PATCH", "/organizations/00010001-0001-0001-0001-000100010002/user_secrets", func(req *http.Request) (int, string) {
		return 400, "something-happened"
	})
	suite.secretsMock.RegisterWithResponseBody("GET", "/definitions/00010001-0001-0001-0001-000100010001", 200, "[]")

	suite.promptMock.OnMethod("InputSecret").Once().Return(expectedValue, nil)
	expanderFn := suite.prepareWorkingExpander()
	expandedValue, err := expanderFn(project.ProjectCategory, secretName, false, suite.project)

	suite.Require().NotNil(err)
	suite.Zero(expandedValue)
}

func (suite *VarPromptingExpanderTestSuite) assertExpansionSaveSuccess(secretName string, category string, expectedValue string) {
	var userChanges []*secretsModels.UserSecretChange
	var bodyErr error
	suite.secretsMock.RegisterWithResponder("PATCH", "/organizations/00010001-0001-0001-0001-000100010002/user_secrets", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &userChanges)
		return 204, "empty-response"
	})
	suite.secretsMock.RegisterWithResponseBody("GET", "/definitions/00010001-0001-0001-0001-000100010001", 200, "[]")

	suite.promptMock.OnMethod("InputSecret").Once().Return(expectedValue, nil)
	expanderFn := suite.prepareWorkingExpander()
	expandedValue, failure := expanderFn(category, secretName, false, suite.project)

	suite.Require().NoError(bodyErr)
	suite.Require().Nil(failure)
	suite.Equal(expectedValue, expandedValue)

	_, failure = expanderFn(category, secretName, false, suite.project)
	suite.Require().Nil(failure, "Should not prompt again because it should have stored/cached the secret")

	suite.Require().Len(userChanges, 1)

	change := userChanges[0]
	suite.Equal(secretName, *change.Name)

	if category == project.ProjectCategory {
		suite.Equal(false, *change.IsUser)
	} else {
		suite.Equal(true, *change.IsUser)
	}

	suite.Equal(strfmt.UUID("00010001-0001-0001-0001-000100010001"), change.ProjectID)

	kp, _ := keypairs.LoadWithDefaults()
	decryptedBytes, failure := kp.DecodeAndDecrypt(*change.Value)
	suite.Require().Nil(failure)
	suite.Equal(expectedValue, string(decryptedBytes))
}

func (suite *VarPromptingExpanderTestSuite) TestSavesSecret() {
	suite.assertExpansionSaveSuccess("proj-secret", project.ProjectCategory, "more amazing")
	suite.assertExpansionSaveSuccess("user-secret", project.UserCategory, "more amazing")
}

func (suite *VarPromptingExpanderTestSuite) TestSaveFails_NonProjectLevelSecret() {
	suite.assertExpansionSaveFailure("org-secret", "not so amazing")
}

func (suite *VarPromptingExpanderTestSuite) TestSaveFails_ProjectLevelSecret() {
	suite.assertExpansionSaveFailure("proj-secret", "utterly boring")
}

func Test_SecretsPromptingExpander_TestSuite(t *testing.T) {
	suite.Run(t, new(VarPromptingExpanderTestSuite))
}
