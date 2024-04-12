package integration

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/project"
)

type CommitIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *CommitIntegrationTestSuite) TestCommitManualBuildScriptMod() {
	suite.OnlyRunForTags(tagsuite.Commit, tagsuite.BuildScripts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("config", "set", constants.OptinBuildscriptsConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs(
			"checkout",
			"ActiveState-CLI/Commit-Test-A#7a1b416e-c17f-4d4a-9e27-cbad9e8f5655",
			".",
		),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Checked out", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	proj, err := project.FromPath(ts.Dirs.Work)
	suite.NoError(err, "Error loading project")

	_, err = buildscript.ScriptFromProject(proj)
	suite.Require().NoError(err) // verify validity

	cp = ts.Spawn("commit")
	cp.Expect("No change")
	cp.ExpectExitCode(1)

	_, err = buildscript.ScriptFromProject(proj)
	suite.Require().NoError(err) // verify validity

	scriptPath := filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName)
	data := fileutils.ReadFileUnsafe(scriptPath)
	data = bytes.ReplaceAll(data, []byte("casestyle"), []byte("case"))
	suite.Require().NoError(fileutils.WriteFile(scriptPath, data), "Update buildscript")

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("commit"),
	)
	cp.Expect("successfully created")
	cp.ExpectExitCode(0)
}

func (suite *CommitIntegrationTestSuite) TestCommitAtTimeChange() {
	suite.OnlyRunForTags(tagsuite.Commit, tagsuite.BuildScripts)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("config", "set", constants.OptinBuildscriptsConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/Commit-Test-A#7a1b416e-c17f-4d4a-9e27-cbad9e8f5655", "."),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Checked out", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	proj, err := project.FromPath(ts.Dirs.Work)
	suite.NoError(err, "Error loading project")

	_, err = buildscript.ScriptFromProject(proj)
	suite.Require().NoError(err) // verify validity

	// Update top-level at_time variable.
	dateTime := "2023-03-01T12:34:56.789Z"
	buildScriptFile := filepath.Join(proj.Dir(), constants.BuildScriptFileName)
	contents, err := fileutils.ReadFile(buildScriptFile)
	suite.Require().NoError(err)
	suite.Require().NoError(fileutils.WriteFile(buildScriptFile, bytes.Replace(contents, []byte("2023-06-22T21:56:10.504Z"), []byte(dateTime), 1)))
	suite.Require().Contains(string(fileutils.ReadFileUnsafe(filepath.Join(proj.Dir(), constants.BuildScriptFileName))), dateTime)

	cp = ts.Spawn("commit")
	cp.Expect("successfully created")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("history")
	cp.Expect("Revision")
	revisionTime, err := time.Parse(time.RFC3339, dateTime)
	suite.Require().NoError(err)
	cp.Expect(revisionTime.Format(time.RFC822))
	cp.ExpectExitCode(0)
}

func TestCommitIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CommitIntegrationTestSuite))
}
