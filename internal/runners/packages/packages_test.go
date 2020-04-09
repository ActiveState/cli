package packages

import (
	"fmt"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"

	"github.com/ActiveState/cli/pkg/projectfile"
)

type PkgTestSuite struct {
	suite.Suite
	apiMock   *apiMock.Mock
	authMock  *authMock.Mock
	invMock   *invMock.Mock
	graphMock *graphMock.Mock
	exiter    *exiter.Exiter
}

func (suite *PkgTestSuite) BeforeTest(suiteName, testName string) {
	suite.apiMock = apiMock.Init()
	suite.invMock = invMock.Init()
	suite.authMock = authMock.Init()
	suite.graphMock = graphMock.Init()
	suite.exiter = exiter.New()

	AddCommand.Exiter = suite.exiter.Exit
	UpdateCommand.Exiter = suite.exiter.Exit
	RemoveCommand.Exiter = suite.exiter.Exit

	projectURL := fmt.Sprintf("https://%s/string/string?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	httpmock.Register("PUT", "/vcs/branch/00010001-0001-0001-0001-000100010001")
	suite.authMock.MockLoggedin()
	suite.invMock.MockIngredientsByName()
	suite.apiMock.MockCommit()
	suite.graphMock.ProjectByOrgAndName(graphMock.NoOptions)
	suite.graphMock.Checkpoint(graphMock.NoOptions)
}

func (suite *PkgTestSuite) AfterTest(suiteName, testName string) {
	suite.invMock.Close()
	suite.apiMock.Close()
	suite.authMock.Close()
	suite.graphMock.Close()

	UpdateArgs.Name = ""

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})
}

func (suite *PkgTestSuite) runsCommand(cmdArgs []string, expectExitCode int, expectOutput string) {
	Cc := Command.GetCobraCmd()
	Cc.SetArgs(cmdArgs)

	out := capturer.CaptureOutput(func() {
		code := suite.exiter.WaitForExit(func() {
			suite.NoError(Cc.Execute())
		})
		suite.Equal(expectExitCode, code, fmt.Sprintf("Expects exit code %d", expectExitCode))
	})
	fmt.Println(out)

	suite.Contains(out, expectOutput)
}
