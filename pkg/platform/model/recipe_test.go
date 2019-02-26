package model_test

import (
	"testing"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/pkg/platform/api/models"

	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/stretchr/testify/suite"
)

type RecipeTestSuite struct {
	suite.Suite
	invMock  *invMock.Mock
	apiMock  *apiMock.Mock
	authMock *authMock.Mock
}

func (suite *RecipeTestSuite) BeforeTest(suiteName, testName string) {
	suite.invMock = invMock.Init()
	suite.apiMock = apiMock.Init()
	suite.authMock = authMock.Init()
}

func (suite *RecipeTestSuite) AfterTest(suiteName, testName string) {
	suite.invMock.Close()
	suite.apiMock.Close()
	suite.authMock.Close()
}

func (suite *RecipeTestSuite) TestGetRecipe() {
	suite.authMock.MockLoggedin()
	suite.apiMock.MockVcsGetCheckpoint()
	suite.invMock.MockOrderRecipes()

	uid := strfmt.UUID("00010001-0001-0001-0001-000100010001")
	platforms, fail := model.FetchRecipesForProject(&models.Project{
		Branches: models.Branches{
			&models.Branch{
				BranchID: uid,
				Default:  true,
				CommitID: &uid,
			},
		},
	})
	suite.Require().NoError(fail.ToError())
	suite.NotEmpty(platforms, "Returns platforms")
}

func TestRecipeSuite(t *testing.T) {
	suite.Run(t, new(RecipeTestSuite))
}
