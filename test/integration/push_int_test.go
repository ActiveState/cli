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
	cp.ExpectLongString(fmt.Sprintf("Project created at https://%s/%s/%s with language %s", constants.PlatformURL, username, pname, "python3"))
	cp.ExpectExitCode(0)

	// Check that languages were reset
	pjfilepath := filepath.Join(ts.Dirs.Work, namespace, constants.ConfigFileName)
	pjfile, fail := projectfile.Parse(pjfilepath)
	suite.Require().NoError(fail.ToError())
	if pjfile.Languages != nil {
		suite.FailNow("Expected languages to be nil, but got: %v", pjfile.Languages)
	}
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
	cp.Expect(fmt.Sprintf("The project %s/%s already exists", username, "Python3"))
	cp.ExpectExitCode(1)
}

func TestPushIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PushIntegrationTestSuite))
}
