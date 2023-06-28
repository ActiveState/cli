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
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type PullIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PullIntegrationTestSuite) TestPull() {
	suite.OnlyRunForTags(tagsuite.Pull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(`project: "https://platform.activestate.com/ActiveState-CLI/Python3"`)

	cp := ts.Spawn("pull")
	cp.Expect("Operating on project ActiveState-CLI/Python3")
	cp.Expect("activestate.yaml has been updated")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("pull")
	cp.Expect("already up to date")
	cp.ExpectExitCode(0)
}

func (suite *PullIntegrationTestSuite) TestPullSetProject() {
	suite.OnlyRunForTags(tagsuite.Pull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(`project: https://platform.activestate.com/ActiveState-CLI/small-python?commitID=9733d11a-dfb3-41de-a37a-843b7c421db4`)

	// update to related project
	cp := ts.Spawn("pull", "--set-project", "ActiveState-CLI/small-python-fork")
	cp.Expect("you may lose changes to your project")
	cp.SendLine("n")
	cp.Expect("Pull aborted by user")
	cp.ExpectNotExitCode(0)

	cp = ts.Spawn("pull", "--non-interactive", "--set-project", "ActiveState-CLI/small-python-fork")
	cp.Expect("activestate.yaml has been updated")
	cp.ExpectExitCode(0)
}

func (suite *PullIntegrationTestSuite) TestPullSetProjectUnrelated() {
	suite.OnlyRunForTags(tagsuite.Pull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(`project: "https://platform.activestate.com/ActiveState-CLI/small-python?commitID=9733d11a-dfb3-41de-a37a-843b7c421db4"`)

	cp := ts.Spawn("pull", "--set-project", "ActiveState-CLI/Python3")
	cp.Expect("you may lose changes to your project")
	cp.SendLine("n")
	cp.Expect("Pull aborted by user")
	cp.ExpectNotExitCode(0)

	cp = ts.Spawn("pull", "--non-interactive", "--set-project", "ActiveState-CLI/Python3")
	cp.Expect("Could not detect common parent")
	cp.ExpectExitCode(1)
}

func (suite *PullIntegrationTestSuite) TestPull_Merge() {
	suite.OnlyRunForTags(tagsuite.Pull)
	projectLine := "project: https://platform.activestate.com/ActiveState-CLI/cli?branch=main&commitID="
	unPulledCommit := "882ae76e-fbb7-4989-acc9-9a8b87d49388"

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	wd := filepath.Join(ts.Dirs.Work, "cli")
	pjfilepath := filepath.Join(ts.Dirs.Work, "cli", constants.ConfigFileName)
	err := fileutils.WriteFile(pjfilepath, []byte(projectLine+unPulledCommit))
	suite.Require().NoError(err)

	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(e2e.OptArgs("push"), e2e.OptWD(wd))
	cp.Expect("Your project has new changes available")
	cp.ExpectExitCode(1)

	cp = ts.SpawnWithOpts(e2e.OptArgs("pull"), e2e.OptWD(wd))
	cp.Expect("Merging history")
	cp.ExpectExitCode(0)

	exe := ts.ExecutablePath()
	if runtime.GOOS == "windows" {
		wd = filepath.ToSlash(wd)
		exe = filepath.ToSlash(exe)
	}
	cp = ts.SpawnCmd("bash", "-c", fmt.Sprintf("cd %s && %s history | head -n 10", wd, exe))
	cp.Expect("Merged")
	cp.ExpectExitCode(0)
}

func (suite *PullIntegrationTestSuite) TestPull_RestoreNamespace() {
	suite.OnlyRunForTags(tagsuite.Pull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(`project: https://platform.activestate.com/ActiveState-CLI/small-python?commitID=9733d11a-dfb3-41de-a37a-843b7c421db4`)

	// Attempt to update to unrelated project.
	cp := ts.Spawn("pull", "--non-interactive", "--set-project", "ActiveState-CLI/Python3")
	cp.Expect("Could not detect common parent")
	cp.ExpectNotExitCode(0)

	// Verify namespace is unchanged.
	cp = ts.Spawn("show")
	cp.Expect("ActiveState-CLI/small-python")
	cp.ExpectExitCode(0)
}

func TestPullIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PullIntegrationTestSuite))
}
