package activate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"fmt"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	promptMock "github.com/ActiveState/cli/internal/prompt/mock"
	"github.com/ActiveState/cli/pkg/platform/api"
	apiMock "github.com/ActiveState/cli/pkg/platform/api/mono/mock"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	rMock "github.com/ActiveState/cli/pkg/platform/runtime/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
)

const ProjectNamespace = "string/string"

type ActivateTestSuite struct {
	suite.Suite
	authMock   *authMock.Mock
	apiMock    *apiMock.Mock
	rMock      *rMock.Mock
	promptMock *promptMock.Mock
	dir        string
	origDir    string
}

func (suite *ActivateTestSuite) SetupSuite() {
	if os.Getenv("CI") == "true" {
		os.Setenv("SHELL", "/bin/bash")
	}

	authMock.Init().MockLoggedin()
}

func (suite *ActivateTestSuite) BeforeTest(suiteName, testName string) {
	suite.authMock = authMock.Init()
	suite.apiMock = apiMock.Init()
	suite.rMock = rMock.Init()
	suite.promptMock = promptMock.Init()
	prompter = suite.promptMock

	var err error

	suite.origDir, err = os.Getwd()
	suite.Require().NoError(err)
	suite.dir, err = ioutil.TempDir("", "activate-test")
	suite.Require().NoError(err)

	err = os.Chdir(suite.dir)
	suite.Require().NoError(err)

	// For some reason the working directory looks different once you cd into it (on mac), so ensure we use the right version
	suite.dir, err = os.Getwd()
	suite.Require().NoError(err)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})

	Args.Namespace = ""

	failures.ResetHandled()
}

func (suite *ActivateTestSuite) AfterTest(suiteName, testName string) {
	os.Chdir(suite.origDir)

	suite.authMock.Close()
	suite.apiMock.Close()
	suite.rMock.Close()
	suite.promptMock.Close()
	err := os.RemoveAll(suite.dir)
	if err != nil {
		fmt.Printf("WARNING: Could not remove temp dir: %s, error: %v", suite.dir, err)
	}
}

func (suite *ActivateTestSuite) TestExecute() {
	suite.rMock.MockFullRuntime()

	err := os.Chdir(filepath.Join(environment.GetRootPathUnsafe(), "state", "activate", "testdata"))
	suite.Require().NoError(err, "unable to chdir to testdata dir")

	Command.Execute()

	suite.Equal(true, true, "Execute didn't panic")
	suite.NoError(failures.Handled(), "No failure occurred")
}

func (suite *ActivateTestSuite) testExecuteWithNamespace(withLang bool) *projectfile.Project {
	suite.rMock.MockFullRuntime()

	if !withLang {
		suite.apiMock.MockGetProjectNoLanguage()
		suite.apiMock.MockVcsGetCheckpointCustomReq(nil)
	}

	targetDir := filepath.Join(suite.dir, ProjectNamespace)
	suite.promptMock.OnMethod("Input").Return(targetDir, nil)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{ProjectNamespace})
	err := Command.Execute()
	suite.Require().NoError(err)

	suite.Equal(true, true, "Execute didn't panic")
	suite.NoError(failures.Handled(), "No failure occurred")

	configFile := filepath.Join(targetDir, constants.ConfigFileName)
	suite.FileExists(configFile)
	pjfile, fail := projectfile.Parse(configFile)
	suite.Require().NoError(fail.ToError())

	return pjfile
}

func (suite *ActivateTestSuite) TestExecuteWithNamespace() {
	suite.testExecuteWithNamespace(false)
}

func (suite *ActivateTestSuite) TestActivateFromNamespaceDontUseExisting() {
	suite.rMock.MockFullRuntime()
	suite.apiMock.MockGetProjectNoLanguage()
	suite.apiMock.MockVcsGetCheckpointCustomReq(nil)

	targetDirOrig := filepath.Join(suite.dir, ProjectNamespace)
	suite.promptMock.OnMethod("Input").Once().Return(targetDirOrig, nil)

	// Set up first checkout
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{ProjectNamespace})
	err := Command.Execute()
	suite.Require().NoError(err)

	suite.FileExists(filepath.Join(targetDirOrig, constants.ConfigFileName))
	savePathForNamespace(ProjectNamespace, targetDirOrig)

	// Now set up the second
	targetDirNew, err := ioutil.TempDir("", "DontUseExisting")
	suite.Require().NoError(err)
	suite.Require().NoError(os.Remove(targetDirNew))

	suite.promptMock.OnMethod("Select").Once().Return("", nil)
	suite.promptMock.OnMethod("Input").Once().Return(targetDirNew, nil)

	err = Command.Execute()
	suite.Require().NoError(err)

	suite.FileExists(filepath.Join(targetDirNew, constants.ConfigFileName))

	os.Chdir(suite.origDir)
	suite.Require().NoError(os.RemoveAll(targetDirNew)) // clean up after
}

func (suite *ActivateTestSuite) TestActivateFromNamespaceInvalidNamespace() {
	fail := activateFromNamespace("foo")
	suite.Equal(failInvalidNamespace.Name, fail.Type.Name)
}

func (suite *ActivateTestSuite) TestActivateFromNamespaceNoProject() {
	suite.authMock.MockLoggedin()
	suite.apiMock.MockGetProject404()

	fail := activateFromNamespace(ProjectNamespace)
	suite.Equal(api.FailProjectNotFound.Name, fail.Type.Name)
}

func TestActivateSuite(t *testing.T) {
	suite.Run(t, new(ActivateTestSuite))
}
