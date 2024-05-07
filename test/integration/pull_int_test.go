package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/analytics/client/sync/reporters"
	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/localcommit"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/project"
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

	suite.assertMergeStrategyNotification(ts, string(types.MergeCommitStrategyFastForward))

	cp = ts.Spawn("pull")
	cp.Expect("already up to date")
	cp.ExpectExitCode(0)
}

func (suite *PullIntegrationTestSuite) TestPull_Merge() {
	suite.OnlyRunForTags(tagsuite.Pull)
	unPulledCommit := "882ae76e-fbb7-4989-acc9-9a8b87d49388"

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/cli", unPulledCommit)

	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(e2e.OptArgs("push"))
	cp.Expect("Your project has new changes available")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	cp = ts.SpawnWithOpts(e2e.OptArgs("pull"))
	cp.Expect("Merging history")
	cp.ExpectExitCode(0)

	exe := ts.ExecutablePath()
	if runtime.GOOS == "windows" {
		exe = filepath.ToSlash(exe)
	}
	cp = ts.SpawnCmd("bash", "-c", fmt.Sprintf("%s history | head -n 10", exe))
	cp.Expect("Merged")
	cp.ExpectExitCode(0)

	suite.assertMergeStrategyNotification(ts, string(types.MergeCommitStrategyRecursiveKeepOnConflict))
}

func (suite *PullIntegrationTestSuite) TestMergeBuildScript() {
	suite.OnlyRunForTags(tagsuite.Pull, tagsuite.BuildScripts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("config", "set", constants.OptinBuildscriptsConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("checkout", "ActiveState-CLI/Merge#447b8363-024c-4143-bf4e-c96989314fdf", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("install", "requests"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Package added", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	proj, err := project.FromPath(ts.Dirs.Work)
	suite.NoError(err, "Error loading project")

	_, err = buildscript.ScriptFromProject(proj)
	suite.Require().NoError(err) // just verify it's a valid build script

	cp = ts.Spawn("pull")
	cp.Expect("The following changes will be merged")
	cp.Expect("requests (2.30.0 → Auto)")
	cp.Expect("Unable to automatically merge build scripts")
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()

	_, err = buildscript.ScriptFromProject(proj)
	suite.Assert().Error(err)
	bytes := fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName))
	suite.Assert().Contains(string(bytes), "<<<<<<<", "No merge conflict markers are in build script")
	suite.Assert().Contains(string(bytes), "=======", "No merge conflict markers are in build script")
	suite.Assert().Contains(string(bytes), ">>>>>>>", "No merge conflict markers are in build script")

	// Verify the local commit was updated to the merge commit.
	// Note: even though the buildscript merge failed, a merge commit was still created. After resolving
	// buildscript conflicts, `state commit` should have something new to commit.
	commit, err := localcommit.Get(ts.Dirs.Work)
	suite.Require().NoError(err)
	suite.Assert().NotEqual(commit.String(), "447b8363-024c-4143-bf4e-c96989314fdf", "localcommit not updated to merged commit")
}

func (suite *PullIntegrationTestSuite) assertMergeStrategyNotification(ts *e2e.Session, strategy string) {
	conflictEvents := filterEvents(parseAnalyticsEvents(suite, ts), func(e reporters.TestLogEntry) bool {
		return e.Category == anaConst.CatInteractions && e.Action == anaConst.ActVcsConflict
	})
	suite.Assert().Equal(1, len(conflictEvents), "Should have a single VCS Conflict event report")
	suite.Assert().Equal(strategy, conflictEvents[0].Label)
}

func (suite *PullIntegrationTestSuite) TestPullNoCommonParent() {
	suite.OnlyRunForTags(tagsuite.Pull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/Python3", "19c5a165-167d-48f1-b5e0-826c2fed6ab7")

	cp := ts.Spawn("pull")
	cp.Expect("Operating on project")
	cp.Expect("ActiveState-CLI/Python3")
	cp.Expect("no common")
	cp.Expect("To review your project history")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func TestPullIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PullIntegrationTestSuite))
}
