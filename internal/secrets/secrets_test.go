package secrets

import (
	"github.com/ActiveState/cli/internal/api"
	"github.com/ActiveState/cli/internal/api/models"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/keypairs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/organizations"
	"github.com/ActiveState/cli/internal/projects"
	secretsapi "github.com/ActiveState/cli/internal/secrets-api"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/secretsapi_test"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
)

type SecretsSuite struct {
	suite.Suite

	projectFile  *projectfile.Project
	organization *models.Organization
	project      *models.Project

	keypair       keypairs.Keypair
	secretsClient *secretsapi.Client
	secretsMock   *httpmock.HTTPMock
	platformMock  *httpmock.HTTPMock
}

func (suite *SecretsSuite) BeforeTest(suiteName, testName string) {
	locale.Set("en-US")
	failures.ResetHandled()

	secretsClient := secretsapi_test.NewDefaultTestClient("bearing123")
	suite.Require().NotNil(secretsClient)
	suite.secretsClient = secretsClient

	var fail *failures.Failure
	suite.keypair, fail = LoadKeypairFromConfigDir()
	suite.Require().NoError(fail.ToError())

	suite.organization, fail = organizations.FetchByURLName("SecretOrg")
	suite.Require().NoError(fail.ToError())

	suite.project, fail = projects.FetchByName("SecretOrg", "SecretProject")
	suite.Require().NoError(fail.ToError())

	suite.secretsMock = httpmock.Activate(secretsClient.BaseURI)
	suite.platformMock = httpmock.Activate(api.Prefix)
}

func (suite *SecretsSuite) AfterTest(suiteName, testName string) {
	httpmock.DeActivate()
	osutil.RemoveConfigFile(constants.KeypairLocalFileName + ".key")
}
