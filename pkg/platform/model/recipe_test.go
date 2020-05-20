package model_test

import (
	"runtime"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/suite"

	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	mono_models "github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/sysinfo"
)

type RecipeTestSuite struct {
	suite.Suite
	invMock     *invMock.Mock
	apiMock     *apiMock.Mock
	authMock    *authMock.Mock
	graphMock   *graphMock.Mock
	platformUID string
}

func (suite *RecipeTestSuite) BeforeTest(suiteName, testName string) {
	suite.invMock = invMock.Init()
	suite.apiMock = apiMock.Init()
	suite.authMock = authMock.Init()
	suite.graphMock = graphMock.Init()

	suite.authMock.MockLoggedin()
	suite.graphMock.Checkpoint(graphMock.NoOptions)
	suite.invMock.MockOrderRecipes()
	suite.invMock.MockPlatforms()

	if runtime.GOOS == "darwin" {
		model.HostPlatform = sysinfo.Linux.String() // mac is not supported yet, so spoof linux
	}

	suite.platformUID = "00010001-0001-0001-0001-000100010001"
	if runtime.GOOS == "windows" {
		suite.platformUID = "00030003-0003-0003-0003-000300030003"
	}
}

func (suite *RecipeTestSuite) AfterTest(suiteName, testName string) {
	suite.invMock.Close()
	suite.apiMock.Close()
	suite.authMock.Close()
	suite.graphMock.Close()
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

func (suite *RecipeTestSuite) TestFetchRecipesForCommit() {
	recipe, fail := model.FetchRawRecipeForCommit("00010001-0001-0001-0001-000100010001", "00010001-0001-0001-0001-000100010001")
	suite.Require().NoError(fail.ToError())
	suite.NotEmpty(recipe, "Returns recipes")
}

func (suite *RecipeTestSuite) TestFetchRecipeForPlatform() {
	recipe, fail := model.FetchRawRecipeForPlatform(suite.mockProject(), model.HostPlatform)
	suite.Require().NoError(fail.ToError())
	suite.NotEmpty(recipe, "Returns recipes")
}

func (suite *RecipeTestSuite) TestFetchRecipeForCommitAndHostPlatform() {
	recipe, fail := model.FetchRawRecipeForCommitAndPlatform("00010001-0001-0001-0001-000100010001", "00010001-0001-0001-0001-000100010001", model.HostPlatform)
	suite.Require().NoError(fail.ToError())
	suite.NotEmpty(recipe, "Returns recipes")
}

func TestRecipeSuite(t *testing.T) {
	suite.Run(t, new(RecipeTestSuite))
}
