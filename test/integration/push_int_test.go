package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/strutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type PushIntegrationTestSuite struct {
	tagsuite.Suite
	username string

	// some variables re-used between tests
	baseProject   string
	language      string
	languageFull  string
	extraPackage  string
	extraPackage2 string
}

func (suite *PushIntegrationTestSuite) SetupSuite() {
	suite.language = "perl"
	suite.languageFull = "perl@5.32.0"
	suite.baseProject = "ActiveState-CLI/Perl-5.32"
	suite.extraPackage = "JSON"
	suite.extraPackage2 = "DateTime"
	if runtime.GOOS == "darwin" {
		suite.language = "python3"
		suite.languageFull = suite.language
		suite.baseProject = "ActiveState-CLI/small-python"
		suite.extraPackage = "six@1.10.0"
	}
}

func (suite *PushIntegrationTestSuite) TestInitAndPush() {
	suite.OnlyRunForTags(tagsuite.Push)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.LoginAsPersistentUser()
	username := "cli-integration-tests"
	pname := strutils.UUID()
	namespace := fmt.Sprintf("%s/%s", username, pname)
	wd := filepath.Join(ts.Dirs.Work, namespace)
	cp := ts.Spawn(
		"init",
		namespace,
		suite.languageFull,
		"--path", wd,
		"--skeleton", "editor",
	)
	cp.ExpectExitCode(0)

	pjfilepath := filepath.Join(ts.Dirs.Work, namespace, constants.ConfigFileName)
	suite.Require().FileExists(pjfilepath)

	cp = ts.SpawnWithOpts(e2e.WithArgs("push"), e2e.WithWorkDirectory(wd))
	cp.Expect("continue?")
	cp.Send("y")
	cp.ExpectLongString("Creating project")
	cp.ExpectLongString("Project created")
	cp.ExpectExitCode(0)

	// Check that languages were reset
	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if pjfile.Languages != nil {
		suite.FailNow("Expected languages to be nil, but got: %v", pjfile.Languages)
	}
	if pjfile.CommitID() == "" {
		suite.FailNow("commitID was not set after running push for project creation")
	}
	if pjfile.BranchName() == "" {
		suite.FailNow("branch was not set after running push for project creation")
	}

	// ensure that we are logged out
	cp = ts.Spawn("auth", "logout")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(e2e.WithArgs("install", suite.extraPackage), e2e.WithWorkDirectory(wd))
	switch runtime.GOOS {
	case "darwin":
		cp.ExpectRe("added|currently building", 60*time.Second) // while cold storage is off
		cp.Wait()
	default:
		cp.Expect("added", 60*time.Second)
		cp.ExpectExitCode(0)
	}

	pjfile, err = projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if !strings.Contains(pjfile.Project, fmt.Sprintf("/%s?", namespace)) {
		suite.FailNow("project field should include project (not headless): " + pjfile.Project)
	}

	ts.LoginAsPersistentUser()

	cp = ts.SpawnWithOpts(e2e.WithArgs("push", namespace), e2e.WithWorkDirectory(wd))
	cp.Expect("Pushing to project")
	cp.ExpectExitCode(0)
}

// Test pushing to a new project from a headless commit
func (suite *PushIntegrationTestSuite) TestPush_HeadlessConvert_NewProject() {
	suite.OnlyRunForTags(tagsuite.Push)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.LoginAsPersistentUser()
	username := "cli-integration-tests"
	pname := strutils.UUID()
	namespace := fmt.Sprintf("%s/%s", username, pname)

	cp := ts.SpawnWithOpts(e2e.WithArgs("install", suite.extraPackage))

	cp.Expect("Multiple Matches")
	cp.Expect("> ")
	// pick the current default selection
	cp.Send("")

	cp.ExpectLongString("An activestate.yaml has been created")
	switch runtime.GOOS {
	case "darwin":
		cp.ExpectRe("added|currently building", 60*time.Second) // while cold storage is off
		cp.Wait()
	default:
		cp.Expect("added", 60*time.Second)
		cp.ExpectExitCode(0)
	}

	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)
	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if !strings.Contains(pjfile.Project, "/commit/") {
		suite.FailNow("project field should be headless but isn't: " + pjfile.Project)
	}

	cp = ts.SpawnWithOpts(e2e.WithArgs("push"))
	cp.ExpectLongString("Who would you like the owner of this project to be?")
	cp.Send("")
	cp.ExpectLongString("What would you like the name of this project to be?")
	cp.SendUnterminated(string([]byte{0033, '[', 'B'})) // move cursor down, and then press enter
	cp.Expect("> Other")
	cp.Send("")
	cp.Expect(">")
	cp.SendLine(pname.String())
	cp.Expect("Project created")
	cp.ExpectExitCode(0)

	pjfile, err = projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if !strings.Contains(pjfile.Project, fmt.Sprintf("/%s?", namespace)) {
		suite.FailNow("project field should include project again: " + pjfile.Project)
	}
}

// Test pushing without permission, and choosing to create a new project
func (suite *PushIntegrationTestSuite) TestPush_NoPermission_NewProject() {
	suite.OnlyRunForTags(tagsuite.Push)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	username := ts.CreateNewUser()
	pname := strutils.UUID()

	cp := ts.SpawnWithOpts(e2e.WithArgs("activate", suite.baseProject, "--path", ts.Dirs.Work))
	cp.ExpectLongString("default project?")
	cp.Send("n")
	cp.Expect("You're Activated", 20*time.Second)
	cp.WaitForInput(10 * time.Second)
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(e2e.WithArgs("install", suite.extraPackage))
	switch runtime.GOOS {
	case "darwin":
		cp.ExpectRe("added|currently building", 60*time.Second) // while cold storage is off
		cp.Wait()
	default:
		cp.Expect("added", 60*time.Second)
		cp.ExpectExitCode(0)
	}

	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)
	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	suite.Require().Contains(pjfile.Project, suite.baseProject)

	cp = ts.SpawnWithOpts(e2e.WithArgs("push"))
	cp.Expect("not authorized")
	cp.Send("y")
	cp.ExpectLongString("Who would you like the owner of this project to be?")
	cp.Send("")
	cp.ExpectLongString("What would you like the name of this project to be?")
	cp.SendUnterminated(string([]byte{0033, '[', 'B'})) // move cursor down, and then press enter
	cp.Expect("> Other")
	cp.Send("")
	cp.Expect(">")
	cp.SendLine(pname.String())
	cp.Expect("Project created")
	cp.ExpectExitCode(0)

	pjfile, err = projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	suite.Require().Contains(pjfile.Project, username)
	suite.Require().Contains(pjfile.Project, pname.String())
}

func (suite *PushIntegrationTestSuite) TestCarlisle() {
	suite.OnlyRunForTags(tagsuite.Push, tagsuite.Carlisle, tagsuite.Headless)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	username := "cli-integration-tests"
	pname := strutils.UUID()
	namespace := fmt.Sprintf("%s/%s", username, pname)

	wd := filepath.Join(ts.Dirs.Work, namespace)
	cp := ts.SpawnWithOpts(
		e2e.WithArgs(
			"activate", suite.baseProject,
			"--path", wd),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.ExpectLongString("default project?")
	cp.Send("n")
	// The activestate.yaml on Windows runs custom activation to set shortcuts and file associations.
	if runtime.GOOS == "windows" {
		cp.Expect("Running Activation Events")
	} else {
		cp.Expect("You're Activated!")
	}
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// ensure that we are logged out
	cp = ts.Spawn("auth", "logout")
	cp.ExpectExitCode(0)

	// anonymous commit
	cp = ts.SpawnWithOpts(e2e.WithArgs(
		"install", suite.extraPackage),
		e2e.WithWorkDirectory(wd),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"))
	switch runtime.GOOS {
	case "darwin":
		cp.ExpectRe("added|currently building", 60*time.Second) // while cold storage is off
		cp.Wait()
	default:
		cp.Expect("added", 60*time.Second)
		cp.ExpectExitCode(0)
	}

	prj, err := project.FromPath(filepath.Join(wd, constants.ConfigFileName))
	suite.Require().NoError(err, "Could not parse project file")
	suite.Assert().False(prj.IsHeadless(), "project should NOT be headless: URL is %s", prj.URL())

	ts.LoginAsPersistentUser()

	cp = ts.SpawnWithOpts(e2e.WithArgs("push", namespace), e2e.WithWorkDirectory(wd))
	cp.ExpectLongString("You are about to create the project")
	cp.Send("y")
	cp.Expect("Project created")
	cp.ExpectExitCode(0)
}

func (suite *PushIntegrationTestSuite) TestPush_Outdated() {
	suite.OnlyRunForTags(tagsuite.Push)
	projectLine := "project: https://platform.activestate.com/ActiveState-CLI/cli?branch=main&commitID="
	unPushedCommit := "882ae76e-fbb7-4989-acc9-9a8b87d49388"

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	wd := filepath.Join(ts.Dirs.Work, namespace)
	pjfilepath := filepath.Join(ts.Dirs.Work, namespace, constants.ConfigFileName)
	err := fileutils.WriteFile(pjfilepath, []byte(projectLine+unPushedCommit))
	suite.Require().NoError(err)

	ts.LoginAsPersistentUser()
	cp := ts.SpawnWithOpts(e2e.WithArgs("push"), e2e.WithWorkDirectory(wd))
	cp.ExpectLongString("Your project has new changes available")
	cp.ExpectExitCode(1)
}

func TestPushIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PushIntegrationTestSuite))
}
