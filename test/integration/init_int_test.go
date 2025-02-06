package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/assets"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/sysinfo"
)

type InitIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *InitIntegrationTestSuite) TestInit() {
	suite.OnlyRunForTags(tagsuite.Init, tagsuite.Critical)
	suite.runInitTest(false, true, "python", "python3")
}

func (suite *InitIntegrationTestSuite) TestInit_Path() {
	suite.OnlyRunForTags(tagsuite.Init)
	suite.runInitTest(true, false, "python", "python3")
}

func (suite *InitIntegrationTestSuite) TestInit_DisambiguatePython() {
	suite.OnlyRunForTags(tagsuite.Init)
	suite.runInitTest(false, false, "python", "python3")
	suite.runInitTest(false, false, "python@3.10.0", "python3")
	if runtime.GOOS != "darwin" {
		// Not supported on mac
		suite.runInitTest(false, false, "python@2.7.18", "python2")
	}
}

func (suite *InitIntegrationTestSuite) TestInit_PartialVersions() {
	suite.OnlyRunForTags(tagsuite.Init)
	suite.runInitTest(false, false, "python@3.10", "python3")
	suite.runInitTest(false, false, "python@3.10.x", "python3")
	suite.runInitTest(false, false, "python@>=3", "python3")
	suite.runInitTest(false, false, "python@2", "python2")
}

func (suite *InitIntegrationTestSuite) runInitTest(addPath bool, sourceRuntime bool, lang string, expectedConfigLanguage string, args ...string) {
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

	if !sourceRuntime {
		cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
		cp.ExpectExitCode(0)
	}

	// Run `state init`, creating the project.
	cp := ts.SpawnWithOpts(e2e.OptArgs(computedArgs...))
	cp.Expect("Initializing Project")
	cp.Expect(fmt.Sprintf("Project '%s' has been successfully initialized", namespace), e2e.RuntimeSourcingTimeoutOpt)
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
	suite.Require().NoError(err)

	content, err := os.ReadFile(configFilepath)
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

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("checkout", "ActiveState-CLI/small-python")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("use", "small-python")
	cp.Expect("Switched to project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	pname := strutils.UUID()
	namespace := fmt.Sprintf("%s/%s", e2e.PersistentUsername, pname)
	cp = ts.Spawn("init", namespace)
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
	suite.OnlyRunForTags(tagsuite.Init, tagsuite.Languages)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.LoginAsPersistentUser()

	// Generate a new namespace for the project to be created.
	pname := strutils.UUID()
	namespace := fmt.Sprintf("%s/%s", e2e.PersistentUsername, pname)

	// Run `state init`, creating the project.
	cp := ts.Spawn("init", namespace, "--language", "python@3.10")
	cp.Expect(fmt.Sprintf("Project '%s' has been successfully initialized", namespace), e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
	ts.NotifyProjectCreated(e2e.PersistentUsername, pname.String())

	// Run `state languages` to verify a full language version was resolved.
	cp = ts.Spawn("languages")
	cp.Expect("python")
	cp.Expect(">=3.10,<3.11 → 3.10.") // note: the patch version is variable, so just expect that it exists
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
	projectName := fmt.Sprintf("test-project-%s", sysinfo.OS().String())

	// First, checkout project to set last used org.
	cp := ts.Spawn("checkout", fmt.Sprintf("%s/Empty", org))
	cp.Expect("Checked out project")

	cp = ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	// Now, run `state init` without specifying the org.
	cp = ts.Spawn("init", projectName, "--language", "python@3")
	cp.Expect(fmt.Sprintf("%s/%s", org, projectName))
	cp.Expect("to track changes for this environment")
	cp.Expect("successfully initialized")
	cp.ExpectExitCode(0)
	ts.NotifyProjectCreated(org, projectName)

	// Verify the config file has the correct project owner.
	suite.Contains(string(fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))), "ActiveState-CLI")
}

func (suite *InitIntegrationTestSuite) TestInit_ChangeSummary() {
	suite.OnlyRunForTags(tagsuite.Init)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("config", "set", "optin.unstable.async_runtime", "true")
	cp.ExpectExitCode(0)

	project := "test-init-change-summary-" + hash.ShortHash(strutils.UUID().String())
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("init", e2e.PersistentUsername+"/"+project, "--language", "python@3.10.10"),
	)
	cp.Expect("Resolving Dependencies")
	cp.Expect("Done")
	ts.NotifyProjectCreated(e2e.PersistentUsername, project)
	cp.Expect("Setting up the following dependencies:")
	cp.Expect("├─ python@3.10.10")
	cp.ExpectExitCode(0)
}

func TestInitIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(InitIntegrationTestSuite))
}
