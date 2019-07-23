package export

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type RecipeCommandTestSuite struct {
	suite.Suite
	apim  *apiMock.Mock
	authm *authMock.Mock
	invm  *invMock.Mock
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

	suite.authm.MockLoggedin()
	suite.apim.MockGetProject()
	suite.apim.MockVcsGetCheckpoint()
	suite.invm.MockPlatforms()
	suite.invm.MockOrderRecipes()
}

func (suite *RecipeCommandTestSuite) AfterTest(suiteName, testName string) {
	suite.invm.Close()
	suite.authm.Close()
	suite.apim.Close()
}

func (suite *RecipeCommandTestSuite) TestExportRecipe() {
	suite.T().Run("with missing commit arg", runRecipeCommandTest(suite))

	cmt := "00020002-0002-0002-0002-000200020002"
	suite.T().Run("with valid commit arg", runRecipeCommandTest(suite, cmt))
}

func runRecipeCommandTest(suite *RecipeCommandTestSuite, args ...string) func(*testing.T) {
	return func(tt *testing.T) {
		// setup "subtest"
		t := suite.T()
		suite.SetT(tt)
		defer suite.SetT(t)

		cc := Command.GetCobraCmd()
		cc.SetArgs(append([]string{"recipe"}, args...))

		projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
		pjfile := projectfile.Project{
			Project: projectURL,
		}
		pjfile.Persist()

		ex := exiter.New()
		Command.Exiter = ex.Exit
		exitCode := ex.WaitForExit(func() {
			Command.Execute()
		})

		suite.Equal(0, exitCode, "exited with code 0")
	}
}

func TestRecipeCommandTestSuite(t *testing.T) {
	suite.Run(t, new(RecipeCommandTestSuite))
}
