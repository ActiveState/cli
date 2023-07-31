package integration

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildscript"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/stretchr/testify/suite"
)

type CommitIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *CommitIntegrationTestSuite) TestCommitManualBuildScriptMod() {
	suite.OnlyRunForTags(tagsuite.Commit)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs(
			"checkout",
			"ActiveState-CLI/Commit-Test-A#7a1b416e-c17f-4d4a-9e27-cbad9e8f5655",
			".",
		),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Checked out")
	cp.ExpectExitCode(0)

	proj, err := project.FromPath(ts.Dirs.Work)
	suite.NoError(err, "Error loading project")

	_, err = buildscript.NewScriptFromProject(proj, nil)
	suite.Require().NoError(err) // verify validity

	cp = ts.Spawn("commit")
	cp.Expect("No change")
	cp.ExpectExitCode(0)

	_, err = buildscript.NewScriptFromProject(proj, nil)
	suite.Require().NoError(err) // verify validity

	scriptPath := filepath.Join(ts.Dirs.Work, constants.BuildScriptFileName)
	data := fileutils.ReadFileUnsafe(scriptPath)
	bytes.ReplaceAll(data, []byte("casestyle"), []byte("case"))
	suite.Require().NoError(fileutils.WriteFile(scriptPath, data), "Update buildscript")

	cp = ts.Spawn("commit")
	cp.Expect("Runtime updated")
	cp.ExpectNotExitCode(0)
}

func TestCommitIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CommitIntegrationTestSuite))
}
