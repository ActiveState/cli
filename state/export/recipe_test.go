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
	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/client/mock"
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
	suite.apim.MockGetProject()
	suite.apim.MockVcsGetCheckpoint()
	suite.invm.MockPlatforms()
	suite.invm.MockOrderRecipes()
	suite.graphMock.ProjectByOrgAndName(graphMock.NoOptions)

	suite.ex = exiter.New()
	Command.Exiter = suite.ex.Exit
}

func (suite *RecipeCommandTestSuite) AfterTest(suiteName, testName string) {
	suite.invm.Close()
	suite.authm.Close()
	suite.apim.Close()
	suite.graphMock.Close()

	RecipeArgs = recipeArgs{}
	RecipeFlags = recipeFlags{}

	cc := Command.GetCobraCmd()
	cc.SetArgs([]string{})

	projectfile.Reset()
	failures.ResetHandled()
}

func (suite *RecipeCommandTestSuite) TestNoArg() {
	suite.runRecipeCommandTest(-1)
}

func (suite *RecipeCommandTestSuite) TestValidArg() {
	cmt := "00020002-0002-0002-0002-000200020002"
	suite.runRecipeCommandTest(-1, cmt)
}

func (suite *RecipeCommandTestSuite) TestValidPlatform() {
	suite.runRecipeCommandTest(-1, "--platform", "linux")
}

func (suite *RecipeCommandTestSuite) TestValidPlatformWithCaps() {
	suite.runRecipeCommandTest(-1, "--platform", "Linux")
}

func (suite *RecipeCommandTestSuite) TestInvalidPlatform() {
	suite.runRecipeCommandTest(1, "--platform", "junk")
}

func (suite *RecipeCommandTestSuite) TestOtherPlatform() {
	suite.runRecipeCommandTest(-1, "--platform", "macos")
}

func (suite *RecipeCommandTestSuite) runRecipeCommandTest(code int, args ...string) {
	cc := Command.GetCobraCmd()
	cc.SetArgs(append([]string{"recipe"}, args...))

	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	exitCode := suite.ex.WaitForExit(func() {
		Command.Execute()
	})

	suite.Equal(code, exitCode, "exited with wrong exitcode")
}

func TestRecipeCommandTestSuite(t *testing.T) {
	suite.Run(t, new(RecipeCommandTestSuite))
}
