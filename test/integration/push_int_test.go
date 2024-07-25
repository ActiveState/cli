package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/termtest"

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
	suite.username = e2e.PersistentUsername
	suite.language = "perl"
	suite.languageFull = "perl@5.32.0"
	suite.baseProject = "ActiveState-CLI/Perl-5.32"
	suite.extraPackage = "JSON"
	suite.extraPackage2 = "DateTime"
	if runtime.GOOS == "darwin" {
		suite.language = "python"
		suite.languageFull = "python"
		suite.baseProject = "ActiveState-CLI/small-python"
		suite.extraPackage = "trender"
	}
}

func (suite *PushIntegrationTestSuite) TestInitAndPush() {
	suite.OnlyRunForTags(tagsuite.Push)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	pname := strutils.UUID()
	namespace := fmt.Sprintf("%s/%s", suite.username, pname)
	cp = ts.SpawnWithOpts(
		e2e.OptArgs(
			"init",
			"--language",
			suite.languageFull,
			namespace,
			".",
		),
	)
	cp.Expect("successfully initialized")
	cp.ExpectExitCode(0)
	ts.NotifyProjectCreated(suite.username, pname.String())

	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)
	suite.Require().FileExists(pjfilepath)

	// Check that languages were reset
	pj, err := project.FromPath(pjfilepath)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(ts.CommitID(), "commitID was not set after running push for project creation")
	suite.Require().NotEmpty(pj.BranchName(), "branch was not set after running push for project creation")

	// ensure that we are logged out
	cp = ts.Spawn("auth", "logout")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", suite.extraPackage)
	switch runtime.GOOS {
	case "darwin":
		cp.ExpectRe("added|being built", termtest.OptExpectTimeout(60*time.Second)) // while cold storage is off
		cp.Wait()
	default:
		cp.Expect("added", termtest.OptExpectTimeout(60*time.Second))
		cp.ExpectExitCode(0)
	}

	pj, err = project.FromPath(pjfilepath)
	suite.Require().NoError(err)
	if !strings.Contains(pj.Source().Project, fmt.Sprintf("/%s?", namespace)) {
		suite.FailNow("project field should include project (not headless): " + pj.Source().Project)
	}

	ts.LoginAsPersistentUser()

	cp = ts.Spawn("push", namespace)
	cp.Expect("Pushing to project")
	cp.ExpectExitCode(0)
}

// Test pushing without permission, and choosing to create a new project
func (suite *PushIntegrationTestSuite) TestPush_NoPermission_NewProject() {
	if runtime.GOOS == "windows" {
		suite.T().Skip("Skipped on Windows for now because SendKeyDown() doesnt work (regardless of bash/cmd)")
	}

	suite.OnlyRunForTags(tagsuite.Push)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	user := ts.CreateNewUser()
	pname := strutils.UUID()

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("checkout", suite.baseProject, ".")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", suite.extraPackage)
	switch runtime.GOOS {
	case "darwin":
		cp.ExpectRe("added|being built", termtest.OptExpectTimeout(60*time.Second)) // while cold storage is off
		cp.Wait()
	default:
		cp.Expect("added", termtest.OptExpectTimeout(60*time.Second))
		cp.ExpectExitCode(0)
	}

	pjfilepath := filepath.Join(ts.Dirs.Work, constants.ConfigFileName)
	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	suite.Require().Contains(pjfile.Project, suite.baseProject)

	cp = ts.Spawn("push")
	cp.Expect("not authorized")
	cp.Expect("(Y/n)")
	cp.SendLine("y")
	cp.Expect("Who would you like the owner of this project to be?")
	cp.SendEnter()
	cp.Expect("What would you like the name of this project to be?")
	cp.SendKeyDown()
	cp.Expect("> Other")
	cp.SendEnter()
	cp.Expect(">")
	cp.SendLine(pname.String())
	cp.Expect("Project created")
	cp.ExpectExitCode(0)
	// Note: no need for ts.NotifyProjectCreated because newly created users and their projects are
	// auto-cleaned by e2e.

	pjfile, err = projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	suite.Require().Contains(pjfile.Project, user.Username)
	suite.Require().Contains(pjfile.Project, pname.String())
}

func (suite *PushIntegrationTestSuite) TestCarlisle() {
	suite.OnlyRunForTags(tagsuite.Push, tagsuite.Carlisle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	pname := strutils.UUID()
	namespace := fmt.Sprintf("%s/%s", suite.username, pname)

	wd := filepath.Join(ts.Dirs.Work, namespace)
	cp := ts.Spawn("activate", suite.baseProject, "--path", wd)
	// The activestate.yaml on Windows runs custom activation to set shortcuts and file associations.
	cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// ensure that we are logged out
	cp = ts.Spawn("auth", "logout")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	// anonymous commit
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("install", suite.extraPackage),
		e2e.OptWD(wd),
	)
	switch runtime.GOOS {
	case "darwin":
		cp.ExpectRe("added|being built", e2e.RuntimeSourcingTimeoutOpt) // while cold storage is off
		cp.Wait()
	default:
		cp.Expect("added", e2e.RuntimeSourcingTimeoutOpt)
		cp.ExpectExitCode(0)
	}

	prj, err := project.FromPath(filepath.Join(wd, constants.ConfigFileName))
	suite.Require().NoError(err, "Could not parse project file")
	suite.Assert().False(prj.IsHeadless(), "project should NOT be headless: URL is %s", prj.URL())

	ts.LoginAsPersistentUser()

	cp = ts.SpawnWithOpts(e2e.OptArgs("push", namespace), e2e.OptWD(wd))
	cp.Expect("Continue? (Y/n)")
	cp.SendLine("y")
	cp.Expect("Project created")
	cp.ExpectExitCode(0)
	ts.NotifyProjectCreated(suite.username, pname.String())
}

func (suite *PushIntegrationTestSuite) TestPush_NoProject() {
	suite.OnlyRunForTags(tagsuite.Push)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("push")
	cp.Expect("No project found")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func (suite *PushIntegrationTestSuite) TestPush_NoAuth() {
	suite.OnlyRunForTags(tagsuite.Push)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	cp := ts.Spawn("push")
	cp.Expect("you need to be authenticated")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func (suite *PushIntegrationTestSuite) TestPush_NoChanges() {
	suite.OnlyRunForTags(tagsuite.Push)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("push")
	cp.Expect("no local changes to push")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func (suite *PushIntegrationTestSuite) TestPush_NameInUse() {
	suite.OnlyRunForTags(tagsuite.Push)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Source project we do not have access to
	ts.PrepareProject("ActiveState-Test-DevNull/push-error-test", "2aa0b8fa-04e2-4079-bde1-d46764e3cb53")

	ts.LoginAsPersistentUser()
	// Target project already exists
	cp := ts.Spawn("push", "-n", "ActiveState-CLI/push-error-test")
	cp.Expect("already in use")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func (suite *PushIntegrationTestSuite) TestPush_Aborted() {
	// Skipped for now due to DX-2244
	suite.T().Skip("Confirming prompt with N not working, must fix first")

	suite.OnlyRunForTags(tagsuite.Push)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Source project we do not have access to
	ts.PrepareProject("ActiveState-Test-DevNull/push-error-test", "2aa0b8fa-04e2-4079-bde1-d46764e3cb53")

	ts.LoginAsPersistentUser()
	// Target project already exists
	cp := ts.Spawn("push")
	cp.Expect("Would you like to create a new project")
	cp.SendLine("n")
	cp.Expect("Project creation aborted by user", termtest.OptExpectTimeout(5*time.Second))
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func (suite *PushIntegrationTestSuite) TestPush_InvalidHistory() {
	suite.OnlyRunForTags(tagsuite.Push)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	// Note the commit we're using here is for another project, in order to repro the error
	ts.PrepareProject("ActiveState-CLI/small-python", "dbc0415e-91e8-407b-ad36-1de0cc5c0cbb")

	ts.LoginAsPersistentUser()
	// Target project already exists
	cp := ts.Spawn("push", "ActiveState-CLI/push-error-test")
	cp.Expect("commit history does not match")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func (suite *PushIntegrationTestSuite) TestPush_PullNeeded() {
	suite.OnlyRunForTags(tagsuite.Push)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/push-error-test", "899c9b4c-d28d-441a-9c28-c84819ba8b1a")

	ts.LoginAsPersistentUser()
	// Target project already exists
	cp := ts.Spawn("push")
	cp.Expect("changes available that need to be merged")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 1 {
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func (suite *PushIntegrationTestSuite) TestPush_Outdated() {
	suite.OnlyRunForTags(tagsuite.Push)
	unPushedCommit := "882ae76e-fbb7-4989-acc9-9a8b87d49388"

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/cli", unPushedCommit)

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("push")
	cp.Expect("Your project has new changes available")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func TestPushIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PushIntegrationTestSuite))
}
