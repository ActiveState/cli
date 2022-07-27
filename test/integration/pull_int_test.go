package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/testsuite"
)

type PullIntegrationTestSuite struct {
	testsuite.Suite
}

func (suite *PullIntegrationTestSuite) TestPull() {
	suite.OnlyRunForTags(testsuite.TagPull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(`project: "https://platform.activestate.com/ActiveState-CLI/Python3"`)

	cp := ts.Spawn("pull")
	cp.Expect("activestate.yaml has been updated")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("pull")
	cp.Expect("already up to date")
	cp.ExpectExitCode(0)
}

func (suite *PullIntegrationTestSuite) TestPullSetProject() {
	suite.OnlyRunForTags(testsuite.TagPull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(`project: https://platform.activestate.com/ActiveState-CLI/small-python?commitID=9733d11a-dfb3-41de-a37a-843b7c421db4`)

	// update to related project
	cp := ts.Spawn("pull", "--set-project", "ActiveState-CLI/small-python-fork")
	cp.ExpectLongString("you may lose changes to your project")
	cp.SendLine("n")
	cp.Expect("Pull aborted by user")
	cp.ExpectNotExitCode(0)

	cp = ts.Spawn("pull", "--non-interactive", "--set-project", "ActiveState-CLI/small-python-fork")
	cp.Expect("activestate.yaml has been updated")
	cp.ExpectExitCode(0)
}

func (suite *PullIntegrationTestSuite) TestPullSetProjectUnrelated() {
	suite.OnlyRunForTags(testsuite.TagPull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(`project: "https://platform.activestate.com/ActiveState-CLI/small-python?commitID=9733d11a-dfb3-41de-a37a-843b7c421db4"`)

	cp := ts.Spawn("pull", "--set-project", "ActiveState-CLI/Python3")
	cp.ExpectLongString("you may lose changes to your project")
	cp.SendLine("n")
	cp.Expect("Pull aborted by user")
	cp.ExpectNotExitCode(0)

	cp = ts.Spawn("pull", "--non-interactive", "--set-project", "ActiveState-CLI/Python3")
	cp.Expect("Could not detect common parent")
	cp.ExpectExitCode(1)
}

func (suite *PullIntegrationTestSuite) TestPull_Merge() {
	suite.OnlyRunForTags(testsuite.TagPull)
	projectLine := "project: https://platform.activestate.com/ActiveState-CLI/cli?branch=main&commitID="
	unPulledCommit := "882ae76e-fbb7-4989-acc9-9a8b87d49388"

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	wd := filepath.Join(ts.Dirs.Work, namespace)
	pjfilepath := filepath.Join(ts.Dirs.Work, namespace, constants.ConfigFileName)
	err := fileutils.WriteFile(pjfilepath, []byte(projectLine+unPulledCommit))
	suite.Require().NoError(err)

	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(e2e.WithArgs("push"), e2e.WithWorkDirectory(wd))
	cp.ExpectLongString("Your project has new changes available")
	cp.ExpectExitCode(1)

	cp = ts.SpawnWithOpts(e2e.WithArgs("pull"), e2e.WithWorkDirectory(wd))
	cp.ExpectLongString("Merging history")
	cp.ExpectExitCode(0)

	exe := ts.ExecutablePath()
	if runtime.GOOS == "windows" {
		wd = filepath.ToSlash(wd)
		exe = filepath.ToSlash(exe)
	}
	cp = ts.SpawnCmd("bash", "-c", fmt.Sprintf("cd %s && %s history | head -n 10", wd, exe))
	cp.ExpectLongString("Merged")
	cp.ExpectExitCode(0)
}

func TestPullIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PullIntegrationTestSuite))
}
