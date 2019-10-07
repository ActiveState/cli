package activate

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/kami-zh/go-capturer"
)

func (suite *ActivateTestSuite) TestActivateNew() {
	suite.rMock.MockFullRuntime()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/organizations")
	httpmock.Register("POST", "organizations/test-owner/projects")
	setupProjectMock()
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

func setupProjectMock() {
	orgProjMockCalled := false //  The project response changes once the project is created so we need
	// to provide a different response after the first call to this mock
	getResponseFile := func(method string, code int, responseFile string, responsePath string) string {
		responseFile = fmt.Sprintf("%s-%s", strings.ToUpper(method), strings.TrimPrefix(responseFile, "/"))
		if code != 200 {
			responseFile = fmt.Sprintf("%s-%d", responseFile, code)
		}
		ext := ".json"
		if filepath.Ext(responseFile) != "" {
			ext = ""
		}
		responseFile = filepath.Join(responsePath, responseFile) + ext

		return responseFile
	}
	responsePath := filepath.Join(environment.GetRootPathUnsafe(), "state", "activate", "testdata", "httpresponse")
	request := "organizations/test-owner/projects/test-name"
	pathToFileWithCommit := "organizations/test-owner/projects/test-name-commit"
	method := "GET"
	code := 200
	httpmock.RegisterWithResponderBody(method, request, code, func(req *http.Request) (int, string) {
		responseFile := getResponseFile(method, code, pathToFileWithCommit, responsePath)
		if !orgProjMockCalled {
			orgProjMockCalled = true
			responseFile = getResponseFile(method, code, request, responsePath)
		}
		return 200, string(fileutils.ReadFileUnsafe(responseFile))
	})
}

func (suite *ActivateTestSuite) TestActivateCopy() {
	suite.rMock.MockFullRuntime()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "/login")
	httpmock.Register("GET", "/organizations")
	httpmock.Register("POST", "organizations/test-owner/projects")
	httpmock.RegisterWithCode("GET", "organizations/ActiveState/projects/CodeIntel", 404)
	setupProjectMock()
	httpmock.Register("POST", "vcs/commit")
	httpmock.Register("PUT", "vcs/branch/00010001-0001-0001-0001-000100010001")
	suite.apiMock.MockGetProject404()

	authentication.Get().AuthenticateWithToken("")

	suite.promptMock.OnMethod("Confirm").Once().Return(true, nil)
	suite.promptMock.OnMethod("Input").Once().Return("test-name", nil)
	suite.promptMock.OnMethod("Select").Once().Return("Python 3", nil)
	suite.promptMock.OnMethod("Input").Once().Return("test-owner", nil)

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
	newURL := "https://platform.activestate.com/test-owner/test-name"
	suite.Equal(newURL, prj.URL())
	suite.Equal("master", prj.Version())
}

func (suite *ActivateTestSuite) TestNewPlatformProject() {
	suite.rMock.MockFullRuntime()
	suite.authMock.MockLoggedin()

	httpmock.Activate(api.GetServiceURL(api.ServiceMono).String())
	defer httpmock.DeActivate()

	httpmock.Register("POST", "organizations/test-owner/projects")
	setupProjectMock()
	httpmock.Register("POST", "vcs/commit")
	httpmock.Register("PUT", "vcs/branch/00010001-0001-0001-0001-000100010001")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--new", "--project", "test-name", "--owner", "test-owner", "--language", "python3"})
	err := Command.Execute()
	suite.NoError(err, "Executed without error")
	suite.NoError(failures.Handled(), "No failure occurred")
}

func (suite *ActivateTestSuite) TestNewPlatformProject_MissingOwner() {
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--new", "--project", "test-name", "--language", "python3"})

	ex := exiter.New()
	Command.Exiter = ex.Exit
	stderr := capturer.CaptureStderr(func() {
		code := ex.WaitForExit(func() {
			Command.Execute()
		})
		suite.Require().Equal(1, code, "Exits with code 1")
	})
	suite.Contains(stderr, locale.T("error_state_activate_owner_flag_not_set"))
}

func (suite *ActivateTestSuite) TestNewPlatformProject_MissingProject() {
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--new", "--owner", "test-owner", "--language", "python3"})

	ex := exiter.New()
	Command.Exiter = ex.Exit
	stderr := capturer.CaptureStderr(func() {
		code := ex.WaitForExit(func() {
			Command.Execute()
		})
		suite.Require().Equal(1, code, "Exits with code 1")
	})
	suite.Contains(stderr, locale.T("error_state_activate_project_flag_not_set"))
}

func (suite *ActivateTestSuite) TestNewPlatformProject_MissingLanguage() {
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--new", "--project", "test-name", "--owner", "test-owner"})

	ex := exiter.New()
	Command.Exiter = ex.Exit
	stderr := capturer.CaptureStderr(func() {
		code := ex.WaitForExit(func() {
			Command.Execute()
		})
		suite.Require().Equal(1, code, "Exits with code 1")
	})
	suite.Contains(stderr, locale.T("error_state_activate_language_flag_not_set"))
}

func (suite *ActivateTestSuite) TestNewPlatformProject_UnknownLanguage() {
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--new", "--project", "test-name", "--owner", "test-owner", "--language", "unknown"})

	ex := exiter.New()
	Command.Exiter = ex.Exit
	stderr := capturer.CaptureStderr(func() {
		code := ex.WaitForExit(func() {
			Command.Execute()
		})
		suite.Require().Equal(1, code, "Exits with code 1")
	})
	suite.Contains(stderr, locale.T("error_state_activate_language_flag_invalid"))
}
