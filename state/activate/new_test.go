package activate

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	graphMock "github.com/ActiveState/cli/pkg/platform/api/graphql/request/mock"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/stretchr/testify/suite"
)

type ActivateNewTestSuite struct {
	ActivateTestSuite
}

func (suite *ActivateNewTestSuite) setupMocks() {
	suite.rMock.MockFullRuntime()
	gmock := suite.rMock.GraphMock
	gmock.Reset()
	gmock.ProjectByOrgAndNameNoCommits(graphMock.Once)
	gmock.ProjectByOrgAndName(graphMock.NoOptions)
}

func (suite *ActivateNewTestSuite) TestActivateNew() {
	suite.setupMocks()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/organizations")
	httpmock.Register("POST", "organizations/sample-org/projects")
	httpmock.Register("POST", "vcs/commit")
	httpmock.Register("PUT", "vcs/branch/00010001-0001-0001-0001-000100010001")
	httpmock.RegisterWithResponderBody("PUT", "vcs/branch/00010001-0001-0001-0001-000100010003", 0, func(req *http.Request) (int, string) {
		return 200, ""
	})

	authentication.Get().AuthenticateWithToken("")

	suite.promptMock.OnMethod("Input").Once().Return("example-proj", nil)
	suite.promptMock.OnMethod("Select").Once().Return("Python 3", nil)
	suite.promptMock.OnMethod("Select").Once().Return("sample-org", nil)

	err := Command.Execute()
	suite.NoError(err, "Executed without error")
	suite.NoError(failures.Handled(), "No failure occurred")

	_, err = os.Stat(filepath.Join(suite.dir, constants.ConfigFileName))
	suite.NoError(err, "Project was created")
}

func (suite *ActivateNewTestSuite) TestActivateCopy() {
	suite.setupMocks()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/organizations")
	httpmock.Register("POST", "organizations/sample-org/projects")
	httpmock.Register("POST", "vcs/commit")
	httpmock.Register("PUT", "vcs/branch/00010001-0001-0001-0001-000100010001")

	authentication.Get().AuthenticateWithToken("")

	suite.promptMock.OnMethod("Confirm").Once().Return(true, nil)
	suite.promptMock.OnMethod("Input").Once().Return("example-proj", nil)
	suite.promptMock.OnMethod("Select").Once().Return("Python 3", nil)
	suite.promptMock.OnMethod("Input").Once().Return("sample-org", nil)

	projPathOriginal := filepath.Join(environment.GetRootPathUnsafe(), "state", "activate", "testdata", constants.ConfigFileName)
	newPath := filepath.Join(suite.dir, constants.ConfigFileName)
	fail := fileutils.CopyFile(projPathOriginal, newPath)
	suite.NoError(fail.ToError(), "Should not fail to copy file")
	os.Chdir(suite.dir)

	err := Command.Execute()
	suite.NoError(err, "Executed without error")
	suite.NoError(failures.Handled(), "No failure occurred")

	_, err = os.Stat(filepath.Join(suite.dir, constants.ConfigFileName))
	suite.NoError(err, "Project was created")
	prj, fail := project.GetOnce()
	suite.NoError(fail.ToError(), "Should retrieve project")
	newURL := "https://platform.activestate.com/ActiveState/CodeIntel?commitID=00010001-0001-0001-0001-000100010001"
	suite.Equal(newURL, prj.URL())
	suite.Equal("master", prj.Version())
}

func (suite *ActivateNewTestSuite) TestNewPlatformProject() {
	suite.setupMocks()
	suite.authMock.MockLoggedin()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "organizations/sample-org/projects")
	httpmock.Register("POST", "vcs/commit")
	httpmock.RegisterWithResponderBody("PUT", "vcs/branch/00010001-0001-0001-0001-000100010001", 0, func(req *http.Request) (int, string) {
		return 200, ""
	})

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--new", "--project", "example-proj", "--owner", "sample-org", "--language", "python3"})
	err := Command.Execute()
	suite.NoError(err, "Executed without error")
	suite.NoError(failures.Handled(), "No failure occurred")
}

func TestActivateNewTestSuite(t *testing.T) {
	suite.Run(t, new(ActivateNewTestSuite))
}
