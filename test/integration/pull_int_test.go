package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/stretchr/testify/suite"
)

type PullIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PullIntegrationTestSuite) TestPull() {
	suite.OnlyRunForTags(tagsuite.Pull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Python3", "59404293-e5a9-4fd0-8843-77cd4761b5b5")

	cp := ts.Spawn("pull")
	cp.Expect("Operating on project")
	cp.Expect("ActiveState-CLI/Python3")
	cp.Expect("activestate.yaml has been updated")
	cp.ExpectExitCode(0)

	projectConfigDir := filepath.Join(ts.Dirs.Work, constants.ProjectConfigDirName)
	suite.Require().True(fileutils.DirExists(projectConfigDir))
	suite.Assert().True(fileutils.FileExists(filepath.Join(projectConfigDir, constants.CommitIdFileName)))

	cp = ts.Spawn("pull")
	cp.Expect("already up to date")
	cp.ExpectExitCode(0)
}

func (suite *PullIntegrationTestSuite) TestPullSetProject() {
	suite.OnlyRunForTags(tagsuite.Pull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "9733d11a-dfb3-41de-a37a-843b7c421db4")

	// update to related project
	cp := ts.Spawn("pull", "--set-project", "ActiveState-CLI/small-python-fork")
	cp.Expect("Are you sure you want to do this? (y/N)")
	cp.SendLine("n")
	cp.Expect("Pull aborted by user")
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()

	cp = ts.Spawn("pull", "--non-interactive", "--set-project", "ActiveState-CLI/small-python-fork")
	cp.Expect("activestate.yaml has been updated")
	cp.ExpectExitCode(0)
}

func (suite *PullIntegrationTestSuite) TestPullSetProjectUnrelated() {
	suite.OnlyRunForTags(tagsuite.Pull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "9733d11a-dfb3-41de-a37a-843b7c421db4")

	cp := ts.Spawn("pull", "--set-project", "ActiveState-CLI/Python3")
	cp.Expect("Are you sure you want to do this? (y/N)")
	cp.SendLine("n")
	cp.Expect("Pull aborted by user")
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()

	cp = ts.Spawn("pull", "--non-interactive", "--set-project", "ActiveState-CLI/Python3")
	cp.Expect("no common base")
	cp.ExpectExitCode(1)
}

func (suite *PullIntegrationTestSuite) TestPull_Merge() {
	suite.OnlyRunForTags(tagsuite.Pull)
	projectLine := "project: https://platform.activestate.com/ActiveState-CLI/cli"
	unPulledCommit := "882ae76e-fbb7-4989-acc9-9a8b87d49388"

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	wd := filepath.Join(ts.Dirs.Work, "cli")
	pjfilepath := filepath.Join(ts.Dirs.Work, "cli", constants.ConfigFileName)
	err := fileutils.WriteFile(pjfilepath, []byte(projectLine))
	suite.Require().NoError(err)
	commitIdFile := filepath.Join(ts.Dirs.Work, "cli", constants.ProjectConfigDirName, constants.CommitIdFileName)
	err = fileutils.WriteFile(commitIdFile, []byte(unPulledCommit))
	suite.Require().NoError(err)

	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(e2e.OptArgs("push"), e2e.OptWD(wd))
	cp.Expect("Your project has new changes available")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

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

	ts.PrepareProject("ActiveState-CLI/small-python", "9733d11a-dfb3-41de-a37a-843b7c421db4")

	// Attempt to update to unrelated project.
	cp := ts.Spawn("pull", "--non-interactive", "--set-project", "ActiveState-CLI/Python3")
	cp.Expect("no common base")
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()

	// Verify namespace is unchanged.
	cp = ts.Spawn("show")
	cp.Expect("ActiveState-CLI/small-python")
	cp.ExpectExitCode(0)
}

func (suite *PullIntegrationTestSuite) TestMergeBuildScript() {
	suite.OnlyRunForTags(tagsuite.Pull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Merge#447b8363-024c-4143-bf4e-c96989314fdf", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	ts.LoginAsPersistentUser()

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("install", "requests"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Package added", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	proj, err := project.FromPath(ts.Dirs.Work)
	suite.NoError(err, "Error loading project")

	_, err = buildscript.NewScriptFromProject(proj, nil)
	suite.Require().NoError(err) // just verify it's a valid build script

	cp = ts.Spawn("pull")
	cp.Expect("Unable to automatically merge build scripts")
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()

	_, err = buildscript.NewScriptFromProject(proj, nil)
	suite.Assert().Error(err)
	bytes := fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName))
	suite.Assert().Contains(string(bytes), "<<<<<<<", "No merge conflict markers are in build script")
	suite.Assert().Contains(string(bytes), "=======", "No merge conflict markers are in build script")
	suite.Assert().Contains(string(bytes), ">>>>>>>", "No merge conflict markers are in build script")
}

func TestPullIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PullIntegrationTestSuite))
}
