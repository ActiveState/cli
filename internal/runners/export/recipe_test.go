package export

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type RecipeCommandTestSuite struct {
	suite.Suite
	apim      *apiMock.Mock
	authm     *authMock.Mock
	invm      *invMock.Mock
	graphMock *graphMock.Mock
	ex        *exiter.Exiter
}

func (suite *RecipeCommandTestSuite) SetupTest() {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "should detect root path")
	os.Chdir(filepath.Join(root, "test"))
}

func (suite *RecipeCommandTestSuite) BeforeTest(suiteName, testName string) {
	suite.apim = apiMock.Init()
	suite.authm = authMock.Init()
	suite.invm = invMock.Init()
	suite.graphMock = graphMock.Init()

	suite.authm.MockLoggedin()
	suite.invm.MockPlatforms()
	suite.invm.MockOrderRecipes()
	suite.graphMock.ProjectByOrgAndName(graphMock.NoOptions)
	suite.graphMock.Checkpoint(graphMock.NoOptions)
	suite.apim.MockCommitsOrder()
}

func (suite *RecipeCommandTestSuite) AfterTest(suiteName, testName string) {
	suite.invm.Close()
	suite.authm.Close()
	suite.apim.Close()
	suite.graphMock.Close()

	projectfile.Reset()
	failures.ResetHandled()
}

func (suite *RecipeCommandTestSuite) TestNoArg() {
	suite.runRecipeCommandTest(false, &RecipeParams{})
}

func (suite *RecipeCommandTestSuite) TestValidArg() {
	suite.runRecipeCommandTest(false, &RecipeParams{CommitID: "00020002-0002-0002-0002-000200020002"})
}

func (suite *RecipeCommandTestSuite) TestValidPlatform() {
	suite.runRecipeCommandTest(false, &RecipeParams{Platform: "linux"})
}

func (suite *RecipeCommandTestSuite) TestValidPlatformWithCaps() {
	suite.runRecipeCommandTest(false, &RecipeParams{Platform: "Linux"})
}

func (suite *RecipeCommandTestSuite) TestInvalidPlatform() {
	suite.runRecipeCommandTest(true, &RecipeParams{Platform: "junk"})
}

func (suite *RecipeCommandTestSuite) TestOtherPlatform() {
	suite.runRecipeCommandTest(false, &RecipeParams{Platform: "macos"})
}

func (suite *RecipeCommandTestSuite) runRecipeCommandTest(wantErr bool, params *RecipeParams) {
	runner := NewRecipe()

	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	err := runner.Run(params)
	if wantErr {
		suite.Require().Error(err)
	} else {
		suite.Require().NoError(err)
	}
}

func TestRecipeCommandTestSuite(t *testing.T) {
	suite.Run(t, new(RecipeCommandTestSuite))
}
