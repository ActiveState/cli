package scripts

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type EditTestSuite struct {
	suite.Suite
	projectFile    *projectfile.Project
	project        *project.Project
	scriptFile     *scriptfile.ScriptFile
	originalEditor string
	cfg            projectfile.ConfigGetter
}

func (suite *EditTestSuite) BeforeTest(suiteName, testName string) {
	var err error
	suite.cfg, err = config.Get()
	suite.Require().NoError(err)

	suite.projectFile = &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/EditOrg/EditProject?commitID=00010001-0001-0001-0001-000100010001"
constants:
  - name: HELLO
    value: hello
scripts:
  - name: hello
    value: echo hello
  - name: hello-constant
    value: echo $constants.HELLO
  - name: replace
    value: echo replaced
`)

	tempDir := os.TempDir()
	err = os.Chdir(tempDir)
	suite.Require().NoError(err, "should change directories without issue")

	err = yaml.Unmarshal([]byte(contents), suite.projectFile)
	suite.Require().NoError(err, "unexpected error marshalling yaml")

	suite.projectFile.SetPath(filepath.Join(tempDir, "activestate.yaml"))
	err = suite.projectFile.Save(suite.cfg)
	suite.Require().NoError(err, "should be able to save in temp dir")

	suite.project, err = project.New(suite.projectFile, nil)
	suite.Require().NoError(err, "unexpected error creating project")

	suite.originalEditor = os.Getenv("EDITOR")
}

func (suite *EditTestSuite) AfterTest(suiteName, testName string) {
	err := os.Remove(suite.projectFile.Path())
	suite.Require().NoError(err, "unexpected error removing project file")

	if suite.scriptFile != nil {
		suite.scriptFile.Clean()
	}

	os.Setenv("EDITOR", suite.originalEditor)
}

func (suite *EditTestSuite) TestCreateScriptFile() {
	script := suite.project.ScriptByName("hello")

	var err error
	suite.scriptFile, err = createScriptFile(script, false)
	suite.Require().NoError(err, "should create file")
}

func (suite *EditTestSuite) TestCreateScriptFile_Expand() {
	script := suite.project.ScriptByName("hello-constant")

	var err error
	suite.scriptFile, err = createScriptFile(script, true)
	suite.Require().NoError(err, "should create file")

	content, err := fileutils.ReadFile(suite.scriptFile.Filename())
	suite.Require().NoError(err, "unexpected error reading file contents")
	v, err := script.Value()
	suite.Require().NoError(err)
	suite.Equal(v, string(content))
}

func (suite *EditTestSuite) TestGetOpenCmd_EditorSet() {
	expected := "debug"
	if runtime.GOOS == "windows" {
		expected = "debug.exe"
	}

	f, err := os.OpenFile(expected, os.O_CREATE|os.O_EXCL, 0700)
	suite.NoError(err, "should be able to create executable file")
	defer os.Remove(f.Name())

	err = f.Close()
	suite.NoError(err, "could no close file")

	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	wd, err := os.Getwd()
	suite.NoError(err, "could not get current working directory")

	err = os.Setenv("PATH", wd)
	suite.NoError(err, "could not set PATH")

	os.Setenv("EDITOR", expected)

	actual, err := getOpenCmd()
	suite.Require().NoError(err, "could not get open command")
	suite.Equal(expected, actual)
}

func (suite *EditTestSuite) TestGetOpenCmd_EditorSet_NotInPath() {
	os.Setenv("EDITOR", "NotInPath")

	_, err := getOpenCmd()
	suite.Require().Error(err, "should get failure when editor is not in PATH")
}

func (suite *EditTestSuite) TestGetOpenCmd_EditorSet_InvalidFilepath() {
	wd, err := os.Getwd()
	suite.NoError(err, "could not get current working directory")

	executeable := "someExecutable"
	if runtime.GOOS == "windows" {
		executeable = "someExecutable.exe"
	}
	os.Setenv("EDITOR", filepath.Join(wd, executeable))

	_, err = getOpenCmd()
	suite.Require().Error(err, "should get failure when editor in path does not exist")
}

func (suite *EditTestSuite) TestGetOpenCmd_EditorSet_NoExtensionWindows() {
	if runtime.GOOS != "windows" {
		suite.T().Skip("the test for file extensions is only relevant for Windows")
	}

	wd, err := os.Getwd()
	suite.NoError(err, "could not get current working director")

	os.Setenv("EDITOR", filepath.Join(wd, "executable"))

	_, err = getOpenCmd()
	suite.Require().Error(err, "should get failure when editor path does not have extension")
}

func (suite *EditTestSuite) TestGetOpenCmd_EditorNotSet() {
	os.Setenv("EDITOR", "")
	var expected string
	platform := runtime.GOOS
	switch platform {
	case "linux":
		expected = openCmdLin
	case "darwin":
		expected = openCmdMac
	case "windows":
		expected = defaultEditorWin
	}

	actual, err := getOpenCmd()
	if platform == "linux" && err != nil {
		suite.EqualError(err, locale.Tr("error_open_not_installed_lin", openCmdLin))
	} else {
		suite.Require().NoError(err, "could not get open command")
		suite.Equal(expected, actual)
	}
}

func (suite *EditTestSuite) TestNewScriptWatcher() {
	script := suite.project.ScriptByName("hello")

	var err error
	suite.scriptFile, err = createScriptFile(script, false)
	suite.Require().NoError(err, "should create file")

	watcher, err := newScriptWatcher(suite.scriptFile)
	suite.Require().NoError(err, "unexpected error creating script watcher")

	catcher := outputhelper.NewCatcher()
	go watcher.run("hello", catcher.Outputer, suite.cfg, project.Get())

	watcher.done <- true

	select {
	case err = <-watcher.errs:
		suite.Require().NoError(err, "should not get error from running watcher")
	default:
		// Do nothing, test passed
	}
}

func (suite *EditTestSuite) TestUpdateProjectFile() {
	replace := suite.project.ScriptByName("replace")

	var err error
	suite.scriptFile, err = createScriptFile(replace, false)
	suite.Require().NoError(err, "unexpected error creating script file")

	err = updateProjectFile(suite.cfg, project.Get(), suite.scriptFile, "replace")
	suite.Require().NoError(err, "should be able to update script file")

	updatedProject := project.Get()
	v1, err := replace.Value()
	suite.Require().NoError(err)
	v2, err := updatedProject.ScriptByName("replace").Value()
	suite.Require().NoError(err)
	suite.Equal(v1, v2)
}

func TestEditSuite(t *testing.T) {
	suite.Run(t, new(EditTestSuite))
}
