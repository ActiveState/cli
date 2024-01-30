package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
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
	suite.Require().NoError(fileutils.Mkdir(filepath.Join(ts.Dirs.Work, ".git")), "could not mimic this being a git repo")

	// Verify the user is prompted to migrate an unmigrated project.
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("packages"),
		e2e.OptAppendEnv(constants.DisableProjectMigrationPrompt+"=false"),
	)
	cp.Expect("migrate")
	cp.Expect("? (y/N)")
	cp.SendEnter()
	cp.Expect("Understood, you can manually upgrade later")

	// Verify that read-only actions still work for unmigrated projects.
	cp.Expect("pytest")
	cp.ExpectExitCode(0)

	// Verify activestate.yaml remains unchanged
	bytes := fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	suite.Assert().Contains(string(bytes), commitID, "as.yaml was not migrated and still contains commitID")

	// Verify that we have the local commit file, since even when declined we still do a partial migration
	projectConfigDir := filepath.Join(ts.Dirs.Work, constants.ProjectConfigDirName)
	suite.Assert().DirExists(projectConfigDir, ".activestate dir was not created")
	localCommitFile := filepath.Join(projectConfigDir, constants.CommitIdFileName)
	suite.Assert().Equal(commitID, string(fileutils.ReadFileUnsafe(localCommitFile)), "local commit file was not created")
	gitignoreFile := filepath.Join(ts.Dirs.Work, ".gitignore")
	suite.Assert().FileExists(gitignoreFile, ".gitignore was not created")

	// Verify no repeat prompts, but we should see a warning
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("packages"),
		e2e.OptAppendEnv(constants.DisableProjectMigrationPrompt+"=false"),
	)
	cp.Expect("outdated format")
	cp.Expect("pytest")
	cp.ExpectExitCode(0)

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

	// Delete local commit file so we can try the prompt again
	suite.Require().NoError(os.Remove(localCommitFile))

	// Verify that migration works.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("packages"),
		e2e.OptAppendEnv(constants.DisableProjectMigrationPrompt+"=false"),
	)
	cp.Expect("migrate")
	cp.Expect("? (y/N)")
	cp.SendLine("Y")
	cp.Expect("success")
	cp.Expect("pytest")
	cp.ExpectExitCode(0)

	// Verify activestate.yaml no longer has commitID
	bytes = fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	suite.Assert().NotContains(string(bytes), "&commitID=", "as.yaml was migrated and does not still contain commitID")
	suite.Assert().NotContains(string(bytes), "migrate-to-buildscripts", "should not have created migration script")
}

func (suite *ProjectMigrationIntegrationTestSuite) TestScriptMigration() {
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Migrations)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	commitID := "c2b3f176-4788-479c-aad3-8359d28ba3ce"
	ts.PrepareActiveStateYAML(`project: https://platform.activestate.com/ActiveState-CLI/Python3-Alternative?commitID=` + commitID)
	suite.Require().NoError(fileutils.Mkdir(filepath.Join(ts.Dirs.Work, ".git")), "could not mimic this being a git repo")

	// Verify the user is prompted to migrate an unmigrated project.
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("packages"),
		e2e.OptAppendEnv(constants.DisableProjectMigrationPrompt+"=false"),
	)
	cp.Expect("migrate")
	cp.Expect("? (y/N)")
	cp.SendEnter()
	cp.Expect("Understood, you can manually upgrade later")

	// Verify that read-only actions still work for unmigrated projects.
	cp.Expect("pytest")
	cp.ExpectExitCode(0)

	// Verify activestate.yaml remains unchanged
	bytes := fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	suite.Assert().Contains(string(bytes), commitID, "as.yaml was not migrated and still contains commitID")

	// Verify activestate.yaml has migration script
	suite.Require().Contains(string(bytes), "migrate-to-buildscripts", "as.yaml has migrate-to-buildscripts script: %s", string(bytes))

	// Verify that we have the local commit file, since even when declined we still do a partial migration
	projectConfigDir := filepath.Join(ts.Dirs.Work, constants.ProjectConfigDirName)
	suite.Assert().DirExists(projectConfigDir, ".activestate dir was not created")
	localCommitFile := filepath.Join(projectConfigDir, constants.CommitIdFileName)
	suite.Assert().Equal(commitID, string(fileutils.ReadFileUnsafe(localCommitFile)), "local commit file was not created")
	gitignoreFile := filepath.Join(ts.Dirs.Work, ".gitignore")
	suite.Assert().FileExists(gitignoreFile, ".gitignore was not created")

	// Verify that script migration works.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("run", "migrate-to-buildscripts"),
		e2e.OptAppendEnv(constants.DisableProjectMigrationPrompt+"=false"),
	)
	cp.Expect("Project successfully migrated")
	cp.ExpectExitCode(0)

	// Verify activestate.yaml no longer has commitID
	bytes = fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	suite.Assert().NotContains(string(bytes), "&commitID=", "as.yaml was migrated and does not still contain commitID")
}

func (suite *ProjectMigrationIntegrationTestSuite) TestScriptMigration_ExistingScripts() {
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Migrations)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	commitID := "c2b3f176-4788-479c-aad3-8359d28ba3ce"
	ts.PrepareActiveStateYAML(fmt.Sprintf(
		`project: https://platform.activestate.com/ActiveState-CLI/Python3-Alternative?commitID=%s
scripts:
  - name: hello
    value: echo hello`, commitID))
	suite.Require().NoError(fileutils.Mkdir(filepath.Join(ts.Dirs.Work, ".git")), "could not mimic this being a git repo")

	// Verify the user is prompted to migrate an unmigrated project.
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("packages"),
		e2e.OptAppendEnv(constants.DisableProjectMigrationPrompt+"=false"),
	)
	cp.Expect("migrate")
	cp.Expect("? (y/N)")
	cp.SendEnter()
	cp.Expect("Understood, you can manually upgrade later")

	// Verify that read-only actions still work for unmigrated projects.
	cp.Expect("pytest")
	cp.ExpectExitCode(0)

	// Verify activestate.yaml has migration script
	bytes := fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	suite.Require().Contains(string(bytes), "migrate-to-buildscripts", "as.yaml has migrate-to-buildscripts script: %s", string(bytes))

	// Verify that script migration works.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("run", "migrate-to-buildscripts"),
		e2e.OptAppendEnv(constants.DisableProjectMigrationPrompt+"=false"),
	)
	cp.Expect("Project successfully migrated")
	cp.ExpectExitCode(0)
}

func TestProjectMigrationIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectMigrationIntegrationTestSuite))
}
