package secrets_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	secretsModels "github.com/ActiveState/cli/internal/secrets-api/models"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/internal/variables"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"
)

type SecretsPromptingExpanderTestSuite struct {
	suite.Suite

	projectFile *projectfile.Project

	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
}

func (suite *SecretsPromptingExpanderTestSuite) BeforeTest(suiteName, testName string) {
	locale.Set("en-US")
	failures.ResetHandled()

	projectFile, err := loadSecretsProject()
	suite.Require().Nil(err, "Unmarshalled project YAML")
	projectFile.Persist()
	suite.projectFile = projectFile

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.Prefix)
}

func (suite *SecretsPromptingExpanderTestSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	projectfile.Reset()
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
}

func (suite *SecretsPromptingExpanderTestSuite) prepareWorkingExpander() variables.ExpanderFunc {
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg", 200)
	suite.platformMock.RegisterWithCode("GET", "/organizations/SecretOrg/projects/SecretProject", 200)

	osutil.CopyTestFileToConfigDir("self-private.key", constants.KeypairLocalFileName+".key", 0600)

	suite.secretsMock.RegisterWithResponder("GET", "/organizations/00010001-0001-0001-0001-000100010002/user_secrets", func(req *http.Request) (int, string) {
		return 200, "user_secrets-empty"
	})
	return secrets.NewPromptingExpander(suite.secretsClient)
}

func (suite *SecretsPromptingExpanderTestSuite) assertExpansionSaveFailure(secretName, expectedValue string, expectedFailureType *failures.FailureType) {
	suite.secretsMock.RegisterWithResponder("PATCH", "/organizations/00010001-0001-0001-0001-000100010002/user_secrets", func(req *http.Request) (int, string) {
		return 400, "something-happened"
	})

	var expandedValue string
	var failure *failures.Failure
	osutil.WrapStdin(func() {
		expanderFn := suite.prepareWorkingExpander()
		expandedValue, failure = expanderFn(secretName, suite.projectFile)
	}, expectedValue)

	suite.Require().NotNil(failure)
	suite.Truef(failure.Type.Matches(expectedFailureType), "unexpected failure type: %v", failure)
	suite.Zero(expandedValue)
}

func (suite *SecretsPromptingExpanderTestSuite) assertExpansionSaveSuccess(secretName, expectedValue string, expectedIsProject, expectedIsUser bool) {
	var userChanges []*secretsModels.UserSecretChange
	var bodyErr error
	suite.secretsMock.RegisterWithResponder("PATCH", "/organizations/00010001-0001-0001-0001-000100010002/user_secrets", func(req *http.Request) (int, string) {
		reqBody, _ := ioutil.ReadAll(req.Body)
		bodyErr = json.Unmarshal(reqBody, &userChanges)
		return 204, "empty-response"
	})

	var expandedValue string
	var failure *failures.Failure
	osutil.WrapStdin(func() {
		expanderFn := suite.prepareWorkingExpander()
		expandedValue, failure = expanderFn(secretName, suite.projectFile)
	}, expectedValue)

	suite.Require().NoError(bodyErr)
	suite.Require().Nil(failure)
	suite.Equal(expectedValue, expandedValue)

	suite.Require().Len(userChanges, 1)

	change := userChanges[0]
	suite.Equal(secretName, *change.Name)
	suite.Equal(expectedIsUser, *change.IsUser)

	if expectedIsProject {
		suite.Equal(strfmt.UUID("00020002-0002-0002-0002-000200020003"), change.ProjectID)
	} else {
		suite.Zero(change.ProjectID)
	}

	kp, _ := keypairs.LoadWithDefaults()
	decryptedBytes, failure := kp.DecodeAndDecrypt(*change.Value)
	suite.Require().Nil(failure)
	suite.Equal(expectedValue, string(decryptedBytes))
}

func (suite *SecretsPromptingExpanderTestSuite) TestSavesOrgLevelSecret() {
	suite.assertExpansionSaveSuccess("org-secret", "amazing", false, false)
}

func (suite *SecretsPromptingExpanderTestSuite) TestSavesProjLevelSecret() {
	suite.assertExpansionSaveSuccess("proj-secret", "more amazing", true, false)
}

func (suite *SecretsPromptingExpanderTestSuite) TestSavesUserLevelSecret() {
	suite.assertExpansionSaveSuccess("user-secret", "user amazing", false, true)
}

func (suite *SecretsPromptingExpanderTestSuite) TestSavesUserProjLevelSecret() {
	suite.assertExpansionSaveSuccess("user-proj-secret", "so amazing", true, true)
}

func (suite *SecretsPromptingExpanderTestSuite) TestSaveFails_NonProjectLevelSecret() {
	suite.assertExpansionSaveFailure("org-secret", "not so amazing", secretsapi.FailSave)
}

func (suite *SecretsPromptingExpanderTestSuite) TestSaveFails_ProjectLevelSecret() {
	suite.assertExpansionSaveFailure("proj-secret", "utterly boring", secretsapi.FailSave)
}

func Test_SecretsPromptingExpander_TestSuite(t *testing.T) {
	suite.Run(t, new(SecretsPromptingExpanderTestSuite))
}
