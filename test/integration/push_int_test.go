package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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
	baseProject  string
	language     string
	extraPackage string
}

func (suite *PushIntegrationTestSuite) SetupSuite() {
	suite.language = "perl@5.32.0"
	suite.baseProject = "ActiveState/Perl-5.32"
	suite.extraPackage = "JSON"
	if runtime.GOOS == "darwin" {
		suite.language = "python3"
		suite.baseProject = "ActiveState-CLI/small-python"
		suite.extraPackage = "datetime"
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
	cp := ts.Spawn(
		"init",
		namespace,
		suite.language,
		"--path", filepath.Join(ts.Dirs.Work, namespace),
		"--skeleton", "editor",
	)
	cp.ExpectExitCode(0)

	wd := filepath.Join(cp.WorkDirectory(), namespace)
	cp = ts.SpawnWithOpts(e2e.WithArgs("push"), e2e.WithWorkDirectory(wd))
	cp.ExpectLongString(fmt.Sprintf("Project created at https://%s/%s/%s", constants.PlatformURL, username, pname))
	cp.ExpectLongString(fmt.Sprintf("with language %s", strings.Split(suite.language, "@")[0]))
	cp.ExpectExitCode(0)

	// Check that languages were reset
	pjfilepath := filepath.Join(ts.Dirs.Work, namespace, constants.ConfigFileName)
	pjfile, err := projectfile.Parse(pjfilepath)
	suite.Require().NoError(err)
	if pjfile.Languages != nil {
		suite.FailNow("Expected languages to be nil, but got: %v", pjfile.Languages)
	}

	// ensure that we are logged out
	cp = ts.Spawn("auth", "logout")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(e2e.WithArgs("install", suite.extraPackage), e2e.WithWorkDirectory(wd))
	cp.Expect("You're about to add packages as an anonymous user")
	cp.Expect("(Y/n)")
	cp.Send("y")
	cp.Expect("added", 30*time.Second)
	cp.ExpectExitCode(0)

	ts.LoginAsPersistentUser()

	cp = ts.SpawnWithOpts(e2e.WithArgs("push"), e2e.WithWorkDirectory(wd))
	cp.Expect("Pushing to project")
	cp.ExpectExitCode(0)
}

func (suite *PushIntegrationTestSuite) TestCarlisle() {
	suite.OnlyRunForTags(tagsuite.Push, tagsuite.Carlisle, tagsuite.Headless)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	username := "cli-integration-tests"
	pname := strutils.UUID()
	namespace := fmt.Sprintf("%s/%s", username, pname)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs(
			"activate", suite.baseProject,
			"--path", filepath.Join(ts.Dirs.Work, namespace)),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.ExpectLongString("default project?")
	cp.Send("n")
	cp.Expect("activated state")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// ensure that we are logged out
	cp = ts.Spawn("auth", "logout")
	cp.ExpectExitCode(0)

	// anonymous commit
	wd := filepath.Join(cp.WorkDirectory(), namespace)
	cp = ts.SpawnWithOpts(e2e.WithArgs("install", suite.extraPackage), e2e.WithWorkDirectory(wd))
	cp.Expect("You're about to add packages as an anonymous user")
	cp.Expect("(Y/n)")
	cp.Send("y")
	cp.Expect("added", 30*time.Second)
	cp.Wait()

	prj, err := project.FromPath(filepath.Join(wd, constants.ConfigFileName))
	suite.Require().NoError(err, "Could not parse project file")
	suite.Assert().True(prj.IsHeadless(), "project should be headless: URL is %s", prj.URL())

	ts.LoginAsPersistentUser()

	// convert to real project
	cp = ts.SpawnWithOpts(e2e.WithArgs("init", namespace), e2e.WithWorkDirectory(wd))
	cp.ExpectLongString("has been successfully initialized")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(e2e.WithArgs("push"), e2e.WithWorkDirectory(wd))
	cp.Expect("Project created")
	cp.ExpectExitCode(0)
}

func (suite *PushIntegrationTestSuite) TestPush_AlreadyExists() {
	suite.OnlyRunForTags(tagsuite.Push)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.LoginAsPersistentUser()
	username := "cli-integration-tests"
	namespace := fmt.Sprintf("%s/%s", username, "Python3")
	cp := ts.Spawn(
		"init",
		namespace,
		"python3",
		"--path", filepath.Join(ts.Dirs.Work, namespace),
		"--skeleton", "editor",
	)
	cp.ExpectExitCode(0)
	wd := filepath.Join(cp.WorkDirectory(), namespace)
	cp = ts.SpawnWithOpts(e2e.WithArgs("push"), e2e.WithWorkDirectory(wd))
	cp.ExpectLongString(fmt.Sprintf("The project %s/%s already exists", username, "Python3"))
	cp.ExpectExitCode(1)
}

func TestPushIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PushIntegrationTestSuite))
}
