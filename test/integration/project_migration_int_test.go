package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
)

type ProjectMigrationIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ProjectMigrationIntegrationTestSuite) TestPromptMigration() {
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Migrations)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	commitID := "c2b3f176-4788-479c-aad3-8359d28ba3ce"
	ts.PrepareActiveStateYAML(`project: https://platform.activestate.com/ActiveState-CLI/Python3-Alternative?commitID=` + commitID)

	projectConfigDir := filepath.Join(ts.Dirs.Work, constants.ProjectConfigDirName)
	localCommitFile := filepath.Join(projectConfigDir, constants.CommitIdFileName)

	// Ensure local commit doesn't exist yet
	suite.Assert().NoFileExists(localCommitFile, "Commit file should not exist yet")

	// Verify migration works without interrupting the user
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("packages"),
	)
	cp.Expect("pytest")
	cp.ExpectExitCode(0)

	// Verify activestate.yaml still has commit
	bytes := fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	suite.Assert().Contains(string(bytes), commitID, "as.yaml was not migrated and still contains commitID")
	// Verify activestate.yaml has migration comment
	suite.Assert().Contains(string(bytes), locale.T("projectmigration_asyaml_comment"), "as.yaml has migration comment")

	// Verify that we have the local commit file
	suite.Assert().DirExists(projectConfigDir, ".activestate dir was not created")
	suite.Assert().Equal(commitID, string(fileutils.ReadFileUnsafe(localCommitFile)), "local commit file was not created")
	gitignoreFile := filepath.Join(ts.Dirs.Work, ".gitignore")
	suite.Assert().FileExists(gitignoreFile, ".gitignore was not created")

	// Make a new commit
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("install", "hello-world"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.ExpectExitCode(0, e2e.RuntimeSourcingTimeoutOpt)

	// Verify both local commit and activestate.yaml are updated
	newCommitID := string(fileutils.ReadFileUnsafe(localCommitFile))
	suite.NotEqual(newCommitID, commitID, "Should have new commit ID")
	pjf, err := projectfile.Parse(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	suite.Require().NoError(err)
	suite.Equal(newCommitID, pjf.LegacyCommitID(), "commit ID in activestate.yaml was not updated")

	// Ensure migration only ran once
	occurrences := 0
	for _, path := range ts.LogFiles() {
		if !strings.HasPrefix(filepath.Base(path), "state-") {
			continue
		}
		contents := string(fileutils.ReadFileUnsafe(path))
		if strings.Contains(contents, "Migrating project to new localcommit format") {
			occurrences++
		}
	}

	suite.Equal(1, occurrences, ts.DebugMessage("Migration ran more than once"))
}

func TestProjectMigrationIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectMigrationIntegrationTestSuite))
}
