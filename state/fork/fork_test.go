package fork

import (
	"testing"

	"github.com/ActiveState/cli/internal/locale"
	promptMock "github.com/ActiveState/cli/internal/prompt/mock"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/pkg/platform/api"
	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/stretchr/testify/suite"
)

const ProjectNamespace = "string/string"
const OrgFlag = "--org"

type ForkTestSuite struct {
	suite.Suite
	authMock   *authMock.Mock
	promptMock *promptMock.Mock
	apiMock    *apiMock.Mock
	graphMock  *graphMock.Mock
}

func (suite *ForkTestSuite) BeforeTest(suiteName, testName string) {
	suite.authMock = authMock.Init()
	suite.promptMock = promptMock.Init()
	suite.apiMock = apiMock.Init()
	suite.graphMock = graphMock.Init()
	prompter = suite.promptMock

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})
}

func (suite *ForkTestSuite) AfterTest(suiteName, testName string) {
	suite.authMock.Close()
	suite.promptMock.Close()
	suite.apiMock.Close()
	suite.graphMock.Close()
}

func (suite *ForkTestSuite) TestExecute() {
	suite.authMock.MockLoggedin()

	suite.apiMock.MockGetOrganizations()
	suite.graphMock.ProjectByOrgAndName(graphMock.NoOptions)
	suite.promptMock.OnMethod("Select").Once().Return("test", nil)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/organizations/test/projects")
	httpmock.Register("PUT", "/vcs/branch/00010001-0001-0001-0001-000100010001")
	httpmock.Register("POST", "/organizations/test/projects/string")

	var execErr error
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{ProjectNamespace})
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = Command.Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)

	suite.Equal(locale.T("state_fork_success", map[string]string{
		"OriginalOwner": "string",
		"OriginalName":  "string",
		"NewOwner":      "test",
		"NewName":       "string",
	})+"\n", outStr)
}

func (suite *ForkTestSuite) TestExecute_OrgFlag() {
	suite.authMock.MockLoggedin()

	suite.apiMock.MockGetOrganizations()
	suite.graphMock.ProjectByOrgAndName(graphMock.NoOptions)
	suite.promptMock.OnMethod("Select").Once().Return("test", nil)

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/organizations/flag-org/projects")
	httpmock.Register("PUT", "/vcs/branch/00010001-0001-0001-0001-000100010001")
	httpmock.Register("POST", "/organizations/flag-org/projects/string")

	var execErr error
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{ProjectNamespace, OrgFlag, "flag-org"})
	outStr, outErr := osutil.CaptureStdout(func() {
		execErr = Command.Execute()
	})
	suite.Require().NoError(outErr)
	suite.Require().NoError(execErr)

	suite.Equal(locale.T("state_fork_success", map[string]string{
		"OriginalOwner": "string",
		"OriginalName":  "string",
		"NewOwner":      "flag-org",
		"NewName":       "string",
	})+"\n", outStr)
}

func TestForkSuite(t *testing.T) {
	suite.Run(t, new(ForkTestSuite))
}
