package scripts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/fileutils"
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
	suite.cfg, err = config.New()
	suite.Require().NoError(err)

	suite.projectFile = &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/EditOrg/EditProject"
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
	suite.Require().NoError(suite.cfg.Close())
}

func (suite *EditTestSuite) TestCreateScriptFile() {
	script, err := suite.project.ScriptByName("hello")
	suite.Require().NoError(err)

	suite.scriptFile, err = createScriptFile(script, false)
	suite.Require().NoError(err, "should create file")
}

func (suite *EditTestSuite) TestCreateScriptFile_Expand() {
	script, err := suite.project.ScriptByName("hello-constant")
	suite.Require().NoError(err)

	suite.scriptFile, err = createScriptFile(script, true)
	suite.Require().NoError(err, "should create file")

	content, err := fileutils.ReadFile(suite.scriptFile.Filename())
	suite.Require().NoError(err, "unexpected error reading file contents")
	v, err := script.Value()
	suite.Require().NoError(err)
	suite.Equal(v, string(content))
}

func (suite *EditTestSuite) TestNewScriptWatcher() {
	script, err := suite.project.ScriptByName("hello")
	suite.Require().NoError(err)

	suite.scriptFile, err = createScriptFile(script, false)
	suite.Require().NoError(err, "should create file")

	watcher, err := newScriptWatcher(suite.scriptFile)
	suite.Require().NoError(err, "unexpected error creating script watcher")

	catcher := outputhelper.NewCatcher()
	proj, err := project.FromWD()
	suite.Require().NoError(err, "unexpected error getting project")
	go watcher.run("hello", catcher.Outputer, suite.cfg, proj)

	watcher.done <- true

	select {
	case err = <-watcher.errs:
		suite.Require().NoError(err, "should not get error from running watcher")
	default:
		// Do nothing, test passed
	}
}

func (suite *EditTestSuite) TestUpdateProjectFile() {
	replace, err := suite.project.ScriptByName("replace")
	suite.Require().NoError(err)

	suite.scriptFile, err = createScriptFile(replace, false)
	suite.Require().NoError(err, "unexpected error creating script file")

	proj, err := project.FromWD()
	suite.Require().NoError(err, "unexpected error getting project")
	err = updateProjectFile(suite.cfg, proj, suite.scriptFile, "replace")
	suite.Require().NoError(err, "should be able to update script file")

	updatedProject, err := project.FromWD()
	suite.Require().NoError(err, "unexpected error getting project")
	v1, err := replace.Value()
	suite.Require().NoError(err)
	script, err := updatedProject.ScriptByName("replace")
	suite.Require().NoError(err)
	v2, err := script.Value()
	suite.Require().NoError(err)
	suite.Equal(v1, v2)
}

func TestEditSuite(t *testing.T) {
	suite.Run(t, new(EditTestSuite))
}
