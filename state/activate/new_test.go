package activate

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
)

func (suite *ActivateTestSuite) TestActivateNew() {
	suite.rMock.MockFullRuntime()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/organizations")
	httpmock.Register("POST", "organizations/test-owner/projects")
	httpmock.Register("GET", "organizations/test-owner/projects/test-name")
	httpmock.Register("POST", "vcs/commit")
	httpmock.Register("PUT", "vcs/branch/00010001-0001-0001-0001-000100010001")

	authentication.Get().AuthenticateWithToken("")

	suite.promptMock.OnMethod("Input").Once().Return("test-name", nil)
	suite.promptMock.OnMethod("Select").Once().Return("Python 3", nil)
	suite.promptMock.OnMethod("Input").Once().Return("test-owner", nil)

	err := Command.Execute()
	suite.NoError(err, "Executed without error")
	suite.NoError(failures.Handled(), "No failure occurred")

	_, err = os.Stat(filepath.Join(suite.dir, constants.ConfigFileName))
	suite.NoError(err, "Project was created")
}
