package integration

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type InitIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *InitIntegrationTestSuite) TestInit() {
	suite.OnlyRunForTags(tagsuite.Init, tagsuite.Critical)
	suite.runInitTest(false, "python", "python3")
}

func (suite *InitIntegrationTestSuite) TestInit_Path() {
	suite.OnlyRunForTags(tagsuite.Init)
	suite.runInitTest(true, "python", "python3")
}

func (suite *InitIntegrationTestSuite) TestInit_DisambiguatePython() {
	suite.OnlyRunForTags(tagsuite.Init)
	suite.runInitTest(false, "python", "python3")
	suite.runInitTest(false, "python@3.10.0", "python3")
	suite.runInitTest(false, "python@2.7.18", "python2")
}

func (suite *InitIntegrationTestSuite) TestInit_PartialVersions() {
	suite.OnlyRunForTags(tagsuite.Init)
	suite.runInitTest(false, "python@3.10", "python3")
	suite.runInitTest(false, "python@3.10.x", "python3")
	suite.runInitTest(false, "python@>=3", "python3")
	suite.runInitTest(false, "python@2", "python2")
}

func (suite *InitIntegrationTestSuite) runInitTest(addPath bool, lang string, expectedConfigLanguage string, args ...string) {
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.LoginAsPersistentUser()

	// Generate a new namespace for the project to be created.
	pname := strutils.UUID()
	namespace := fmt.Sprintf("%s/%s", e2e.PersistentUsername, pname)
	computedArgs := append([]string{"init", "--language", lang, namespace}, args...)
	if addPath {
		computedArgs = append(computedArgs, ts.Dirs.Work)
	}

	// Run `state init`, creating the project.
	cp := ts.Spawn(computedArgs...)
	cp.Expect("Skipping runtime setup")
	cp.Expect(fmt.Sprintf("Project '%s' has been successfully initialized", namespace))
	cp.ExpectExitCode(0)
	ts.NotifyProjectCreated(e2e.PersistentUsername, pname.String())

	// Verify the config template contains the correct shell, language, and content.
	configFilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)
	suite.Require().FileExists(configFilepath)

	templateFile, err := assets.ReadFileBytes("activestate.yaml.python.tpl")
	if err != nil {
		panic(err.Error())
	}
	shell := "bash"
	if runtime.GOOS == "windows" {
		shell = "batch"
	}
	yaml, err := strutils.ParseTemplate(
		string(templateFile),
		map[string]interface{}{
			"Owner":    e2e.PersistentUsername,
			"Project":  pname.String(),
			"Shell":    shell,
			"Language": expectedConfigLanguage,
			"LangExe":  language.MakeByName(expectedConfigLanguage).Executable().Filename(),
		}, nil)

	content, err := ioutil.ReadFile(configFilepath)
	suite.Require().NoError(err)
	suite.Contains(string(content), yaml)
}

func (suite *InitIntegrationTestSuite) TestInit_NoLanguage() {
	suite.OnlyRunForTags(tagsuite.Init)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("init", "test-user/test-project")
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()
}

func (suite *InitIntegrationTestSuite) TestInit_InferLanguageFromUse() {
	suite.OnlyRunForTags(tagsuite.Init)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.LoginAsPersistentUser()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Python3")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("use", "Python3"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Switched to project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	pname := strutils.UUID()
	namespace := fmt.Sprintf("%s/%s", e2e.PersistentUsername, pname)
	cp = ts.Spawn("init", namespace)
	cp.Expect("Skipping runtime setup")
	cp.Expect("successfully initialized")
	cp.ExpectExitCode(0)
	ts.NotifyProjectCreated(e2e.PersistentUsername, pname.String())

	suite.Contains(string(fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))), "language: python3")
}

func (suite *InitIntegrationTestSuite) TestInit_NotAuthenticated() {
	suite.OnlyRunForTags(tagsuite.Init)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("init", "test-user/test-project", "python3")
	cp.Expect("You need to be authenticated to initialize a project.")
}

func (suite *InitIntegrationTestSuite) TestInit_AlreadyExists() {
	suite.OnlyRunForTags(tagsuite.Init)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.LoginAsPersistentUser()

	cp := ts.Spawn("init", fmt.Sprintf("%s/test-project", e2e.PersistentUsername), "--language", "python@3")
	cp.Expect("The project 'test-project' already exists under 'cli-integration-tests'")
	cp.ExpectExitCode(1)
}

func (suite *InitIntegrationTestSuite) TestInit_Resolved() {
	suite.OnlyRunForTags(tagsuite.Init)
	if runtime.GOOS == "darwin" {
		suite.T().Skip("Skipping mac for now as the builds are still too unreliable")
		return
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.LoginAsPersistentUser()

	// Generate a new namespace for the project to be created.
	pname := strutils.UUID()
	namespace := fmt.Sprintf("%s/%s", e2e.PersistentUsername, pname)

	// Run `state init`, creating the project.
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("init", namespace, "--language", "python@3.10"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect(fmt.Sprintf("Project '%s' has been successfully initialized", namespace), e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
	ts.NotifyProjectCreated(e2e.PersistentUsername, pname.String())

	// Run `state languages` to verify a full language version was resolved.
	cp = ts.Spawn("languages")
	cp.Expect("python")
	cp.Expect("3.10 → 3.10.0")
	cp.ExpectExitCode(0)
}

func (suite *InitIntegrationTestSuite) TestInit_NoOrg() {
	suite.OnlyRunForTags(tagsuite.Init)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.LoginAsPersistentUser()

	cp := ts.Spawn("init", "random-org/test-project", "--language", "python@3")
	cp.Expect("The organization 'random-org' either does not exist, or you do not have permissions to create a project in it.")
	cp.ExpectExitCode(1)
}

func (suite *InitIntegrationTestSuite) TestInit_InferredOrg() {
	suite.OnlyRunForTags(tagsuite.Init)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.LoginAsPersistentUser()
	ts.IgnoreLogErrors()

	org := "ActiveState-CLI"
	projectName := "test-project"

	// First, checkout project to set last used org.
	cp := ts.Spawn("checkout", fmt.Sprintf("%s/Python3", org))
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")

	// Now, run `state init` without specifying the org.
	cp = ts.Spawn("init", projectName, "--language", "python@3")
	cp.Expect(fmt.Sprintf("%s/%s", org, projectName))
	cp.Expect("successfully initialized")
	cp.ExpectExitCode(0)
	ts.NotifyProjectCreated(org, projectName)

	// Verify the config file has the correct project owner.
	suite.Contains(string(fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))), "ActiveState-CLI")
}

func TestInitIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InitIntegrationTestSuite))
}
