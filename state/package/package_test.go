package pkg

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	invMock "github.com/ActiveState/cli/pkg/platform/api/inventory/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type PkgTestSuite struct {
	suite.Suite
	apiMock  *apiMock.Mock
	authMock *authMock.Mock
	invMock  *invMock.Mock
	exiter   *exiter.Exiter
}

func (suite *PkgTestSuite) BeforeTest(suiteName, testName string) {
	suite.apiMock = apiMock.Init()
	suite.invMock = invMock.Init()
	suite.authMock = authMock.Init()
	suite.exiter = exiter.New()

	updateProjectMock()

	AddCommand.Exiter = suite.exiter.Exit
	UpdateCommand.Exiter = suite.exiter.Exit
	RemoveCommand.Exiter = suite.exiter.Exit

	projectURL := fmt.Sprintf("https://%s/sample-org/example-proj?commitID=00010001-0001-0001-0001-000100010001", constants.PlatformURL)
	pjfile := projectfile.Project{
		Project: projectURL,
	}
	pjfile.Persist()

	suite.authMock.MockLoggedin()
	suite.invMock.MockIngredientsByName()
	suite.apiMock.MockGetProject()
	suite.apiMock.MockVcsGetCheckpoint()
	suite.apiMock.MockCommit()
}

func updateProjectMock() {
	mp := model.ProjectProviderMock()

	for _, proj := range mp.ProjectsResp.Projects {
		if proj.Name == "example-proj" && proj.OrganizationID == mp.OrgData.ID("sample-org") {
			cid := strfmt.UUID("00010001-0001-0001-0001-000100010001")
			proj.Branches[0].CommitID = &cid
		}
	}
}

func (suite *PkgTestSuite) AfterTest(suiteName, testName string) {
	suite.invMock.Close()
	suite.apiMock.Close()
	suite.authMock.Close()

	UpdateArgs.Name = ""

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})

	model.ResetProviderMock()
}

func (suite *PkgTestSuite) runsCommand(cmdArgs []string, expectExitCode int, expectOutput string) {
	Cc := Command.GetCobraCmd()
	Cc.SetArgs(cmdArgs)

	out := capturer.CaptureOutput(func() {
		code := suite.exiter.WaitForExit(func() {
			suite.Require().NoError(Cc.Execute())
		})
		suite.Require().Equal(expectExitCode, code, fmt.Sprintf("Expects exit code %d", expectExitCode))
	})

	suite.Contains(out, expectOutput)
}
