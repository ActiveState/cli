package integration

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ProjectMigrationIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ProjectMigrationIntegrationTestSuite) TestPromptMigration() {
	suite.OnlyRunForTags(tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	commitID := "9090c128-e948-4388-8f7f-96e2c1e00d98"
	ts.PrepareActiveStateYAML(`project: https://platform.activestate.com/ActiveState-CLI/test?commitID=` + commitID)
	suite.Require().NoError(fileutils.Mkdir(filepath.Join(ts.Dirs.Work, ".git")), "could not mimic this being a git repo")

	// Verify the user is prompted to migrate an unmigrated project.
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("packages"),
		e2e.OptAppendEnv(constants.DisableProjectMigrationPrompt+"=false"),
	)
	cp.Expect("migrate")
	cp.Expect("? (y/N)")
	cp.SendEnter()
	cp.Expect("declined")

	// Verify that read-only actions still work for unmigrated projects.
	cp.Expect("pylint")
	cp.Expect("pytest")
	cp.ExpectExitCode(0)

	// Verify activestate.yaml remains unchanged and a .activestate/commit was not created, nor was a
	// .gitignore created.
	bytes := fileutils.ReadFileUnsafe(filepath.Join(ts.Dirs.Work, constants.ConfigFileName))
	suite.Assert().Contains(string(bytes), commitID, "as.yaml was migrated and does not still contain commitID")
	projectConfigDir := filepath.Join(ts.Dirs.Work, constants.ProjectConfigDirName)
	suite.Assert().NoDirExists(projectConfigDir, ".activestate dir was created")
	gitignoreFile := filepath.Join(ts.Dirs.Work, ".gitignore")
	suite.Assert().NoFileExists(gitignoreFile, ".gitignore was created")

	// Verify that migration works.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("packages"),
		e2e.OptAppendEnv(constants.DisableProjectMigrationPrompt+"=false"),
	)
	cp.Expect("migrate")
	cp.Expect("? (y/N)")
	cp.SendLine("Y")
	cp.Expect("success")

	cp.Expect("pylint")
	cp.Expect("pytest")
	cp.ExpectExitCode(0)

	// Verify .activestate/commit and .gitignore were created.
	suite.Require().True(fileutils.DirExists(projectConfigDir), ",migration should have created "+projectConfigDir)
	commitIDFile := filepath.Join(projectConfigDir, constants.CommitIdFileName)
	suite.Assert().True(fileutils.FileExists(commitIDFile), "commit file not created")
	suite.Assert().Contains(string(fileutils.ReadFileUnsafe(commitIDFile)), commitID, "migration did not populate .activestate/commit")
	suite.Assert().True(fileutils.FileExists(gitignoreFile), "migration did not create .gitignore")
	suite.Assert().Contains(string(fileutils.ReadFileUnsafe(gitignoreFile)), fmt.Sprintf("%s/%s", constants.ProjectConfigDirName, constants.CommitIdFileName), "commit file not added to .gitignore")

	// Verify no prompt for migrated project.
	cp = ts.SpawnWithOpts(
		e2e.OptArgs("packages"),
		e2e.OptAppendEnv(constants.DisableProjectMigrationPrompt+"=false"),
	)
	cp.Expect("pylint")
	cp.Expect("pytest")
	cp.ExpectExitCode(0)
}

func TestProjectMigrationIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ProjectMigrationIntegrationTestSuite))
}
