package model_test

import (
	"testing"

	"github.com/ActiveState/sysinfo"

	"github.com/go-openapi/strfmt"

	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"

	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
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

	suite.authMock.MockLoggedin()
	suite.apiMock.MockVcsGetCheckpoint()
	suite.invMock.MockOrderRecipes()
	suite.invMock.MockPlatforms()

	model.OS = sysinfo.Linux // only linux is supported atm
}

func (suite *RecipeTestSuite) AfterTest(suiteName, testName string) {
	suite.invMock.Close()
	suite.apiMock.Close()
	suite.authMock.Close()
}

func (suite *RecipeTestSuite) mockProject() *mono_models.Project {
	uid := strfmt.UUID("00010001-0001-0001-0001-000100010001")
	return &mono_models.Project{
		Branches: mono_models.Branches{
			&mono_models.Branch{
				BranchID: uid,
				Default:  true,
				CommitID: &uid,
			},
		},
	}
}

func (suite *RecipeTestSuite) TestGetRecipe() {
	recipes, fail := model.FetchRecipesForProject(suite.mockProject())
	suite.Require().NoError(fail.ToError())
	suite.NotEmpty(recipes, "Returns recipes")
}

func (suite *RecipeTestSuite) TestFetchEffectiveRecipeForProject() {
	recipe, fail := model.FetchEffectiveRecipeForProject(suite.mockProject())
	suite.Require().NoError(fail.ToError())
	suite.Equal(strfmt.UUID("00010001-0001-0001-0001-000100010001"), *recipe.PlatformID, "Returns recipe")
}

func (suite *RecipeTestSuite) TestRecipeToBuildRecipe() {
	recipe, fail := model.FetchEffectiveRecipeForProject(suite.mockProject())
	suite.Require().NoError(fail.ToError())
	buildRecipe, fail := model.RecipeToBuildRecipe(recipe)
	suite.Require().NoError(fail.ToError())
	suite.Equal(strfmt.UUID("00010001-0001-0001-0001-000100010001"), *buildRecipe.PlatformID, "Returns recipe")
}

func TestRecipeSuite(t *testing.T) {
	suite.Run(t, new(RecipeTestSuite))
}
