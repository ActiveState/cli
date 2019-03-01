package activate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/pkg/platform/api"
	"gopkg.in/AlecAivazis/survey.v1/terminal"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	projMock "github.com/ActiveState/cli/internal/projects/mock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/assert"
)

const ProjectNamespace = "string/string"

type ActivateTestSuite struct {
	suite.Suite
	authMock *authMock.Mock
	projMock *projMock.Mock
	dir      string
}

func (suite *ActivateTestSuite) SetupSuite() {
	if os.Getenv("CI") == "true" {
		os.Setenv("SHELL", "/bin/bash")
	}

	authMock.Init().MockLoggedin()
}

func (suite *ActivateTestSuite) BeforeTest(suiteName, testName string) {
	suite.authMock = authMock.Init()
	suite.projMock = projMock.Init()

	var err error

	suite.dir, err = ioutil.TempDir("", "activate-test")
	suite.Require().NoError(err)

	err = os.Chdir(suite.dir)
	suite.Require().NoError(err)

	// For some reason the working directory looks different once you cd into it (on mac), so ensure we use the right version
	suite.dir, err = os.Getwd()
	suite.Require().NoError(err)
}

func (suite *ActivateTestSuite) AfterTest(suiteName, testName string) {
	suite.authMock.Close()
	suite.projMock.Close()
	err := os.RemoveAll(suite.dir)
	suite.Require().NoError(err)
}

func (suite *ActivateTestSuite) TestExecute() {
	err := os.Chdir(filepath.Join(environment.GetRootPathUnsafe(), "state", "activate", "testdata"))
	suite.Nil(err, "unable to chdir to testdata dir")

	Command.Execute()

	suite.Equal(true, true, "Execute didn't panic")
	suite.NoError(failures.Handled(), "No failure occurred")
}

func (suite *ActivateTestSuite) TestExecuteWithNamespace() {
	suite.authMock.MockLoggedin()
	suite.projMock.MockGetProject()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{ProjectNamespace})

	osutil.WrapStdinWithDelay(10*time.Millisecond, func() { Command.Execute() }, "")

	suite.Equal(true, true, "Execute didn't panic")
	suite.NoError(failures.Handled(), "No failure occurred")

	suite.FileExists(filepath.Join(suite.dir, ProjectNamespace, constants.ConfigFileName))
}

func (suite *ActivateTestSuite) TestActivateFromNamespace() {
	suite.authMock.MockLoggedin()
	suite.projMock.MockGetProject()

	fail := suite.executeWithInput(ProjectNamespace, "")
	suite.Require().NoError(fail.ToError())
	suite.FileExists(filepath.Join(suite.dir, ProjectNamespace, constants.ConfigFileName))
}

func (suite *ActivateTestSuite) TestActivateFromNamespaceCustomDir() {
	suite.authMock.MockLoggedin()
	suite.projMock.MockGetProject()

	targetDir, err := ioutil.TempDir(suite.dir, "CustomDir")
	suite.Require().NoError(err)
	suite.Require().NoError(os.Remove(targetDir))

	fail := suite.executeWithInput(ProjectNamespace, targetDir)
	suite.Require().NoError(fail.ToError())
	suite.FileExists(filepath.Join(targetDir, constants.ConfigFileName))
}

func (suite *ActivateTestSuite) TestActivateFromNamespaceDontUseExisting() {
	suite.authMock.MockLoggedin()
	suite.projMock.MockGetProject()

	// Set up first checkout
	implicitDir := filepath.Join(suite.dir, ProjectNamespace)
	fail := suite.executeWithInput(ProjectNamespace, "")
	suite.Require().NoError(fail.ToError())
	suite.FileExists(filepath.Join(implicitDir, constants.ConfigFileName))
	savePathForNamespace(ProjectNamespace, implicitDir)

	// Now set up the second
	targetDir, err := ioutil.TempDir(suite.dir, "DontUseExisting")
	suite.Require().NoError(err)
	suite.Require().NoError(os.Remove(targetDir))
	fail = suite.executeWithInput(ProjectNamespace, terminal.KeyArrowDown, "", targetDir)
	suite.Require().NoError(fail.ToError())
	suite.FileExists(filepath.Join(targetDir, constants.ConfigFileName))
}

func (suite *ActivateTestSuite) TestActivateFromNamespaceUseExisting() {
	suite.authMock.MockLoggedin()
	suite.projMock.MockGetProject()

	// Set up first checkout
	implicitDir := filepath.Join(suite.dir, ProjectNamespace)
	fail := suite.executeWithInput(ProjectNamespace, "")
	suite.Require().NoError(fail.ToError())
	suite.FileExists(filepath.Join(implicitDir, constants.ConfigFileName))
	savePathForNamespace(ProjectNamespace, implicitDir)

	os.Chdir(suite.dir)

	fail = suite.executeWithInput(ProjectNamespace, "")
	suite.Require().NoError(fail.ToError())

	wd, err := os.Getwd()
	suite.Require().NoError(err)
	wd, err = filepath.Abs(wd)
	suite.Require().NoError(err)
	suite.Equal(implicitDir, wd)
}

func (suite *ActivateTestSuite) TestActivateFromNamespaceInvalidNamespace() {
	fail := activateFromNamespace("foo")
	suite.Equal(failInvalidNamespace.Name, fail.Type.Name)
}

func (suite *ActivateTestSuite) TestActivateFromNamespaceNoProject() {
	suite.authMock.MockLoggedin()
	suite.projMock.MockGetProject404()

	fail := activateFromNamespace(ProjectNamespace)
	suite.Equal(api.FailProjectNotFound.Name, fail.Type.Name)
}

func (suite *ActivateTestSuite) executeWithInput(namespace string, input ...interface{}) *failures.Failure {
	var fail *failures.Failure
	osutil.WrapStdinWithDelay(10*time.Millisecond, func() { fail = activateFromNamespace(namespace) }, input...)
	return fail
}

func TestActivateSuite(t *testing.T) {
	suite.Run(t, new(ActivateTestSuite))
}
