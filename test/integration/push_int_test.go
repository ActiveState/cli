package integration

import (
	"fmt"
	"path/filepath"
	"testing"

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
		"python3",
		"--path", filepath.Join(ts.Dirs.Work, namespace),
		"--skeleton", "editor",
	)
	cp.ExpectExitCode(0)

	wd := filepath.Join(cp.WorkDirectory(), namespace)
	cp = ts.SpawnWithOpts(e2e.WithArgs("push"), e2e.WithWorkDirectory(wd))
	cp.ExpectLongString(fmt.Sprintf("Project created at https://%s/%s/%s", constants.PlatformURL, username, pname))
	cp.ExpectLongString("with language python3")
	cp.ExpectExitCode(0)

	// Check that languages were reset
	pjfilepath := filepath.Join(ts.Dirs.Work, namespace, constants.ConfigFileName)
	pjfile, fail := projectfile.Parse(pjfilepath)
	suite.Require().NoError(fail.ToError())
	if pjfile.Languages != nil {
		suite.FailNow("Expected languages to be nil, but got: %v", pjfile.Languages)
	}
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
			"activate", "ActiveState-CLI/small-python",
			"--path", filepath.Join(ts.Dirs.Work, namespace)),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("activated state")
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	// ensure that we are logged out
	cp = ts.Spawn("auth", "logout")
	cp.ExpectExitCode(0)

	// anonymous commit
	wd := filepath.Join(cp.WorkDirectory(), namespace)
	cp = ts.SpawnWithOpts(e2e.WithArgs("install", "DateTime"), e2e.WithWorkDirectory(wd))
	cp.Expect("You're about to add packages as an anonymous user")
	cp.Expect("(Y/n)")
	cp.SendCtrlC()
	cp.ExpectExitCode(1)

	prj, fail := project.FromPath(filepath.Join(wd, constants.ConfigFileName))
	suite.Require().NoError(fail.ToError(), "Could not parse project file")
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
