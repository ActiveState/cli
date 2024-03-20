package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/analytics/client/blackhole"
	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	runbitsGit "github.com/ActiveState/cli/internal/runbits/git"
	"github.com/ActiveState/cli/internal/testhelpers/outputhelper"
	"github.com/ActiveState/cli/pkg/project"
)

type GitTestSuite struct {
	suite.Suite
	dir        string
	anotherDir string
}

func (suite *GitTestSuite) BeforeTest(suiteName, testName string) {

	var err error
	suite.dir, err = os.MkdirTemp("", testName)
	suite.NoError(err, "could not create a temporary directory")

	repo, err := git.PlainInit(suite.dir, false)
	suite.NoError(err, "could not init a new git repo")

	worktree, err := repo.Worktree()
	suite.NoError(err, "could not get repository worktree")

	projectURL := fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, "test-owner", "test-project")

	err = fileutils.WriteFile(filepath.Join(suite.dir, "activestate.yaml"), []byte("project: "+projectURL))
	suite.NoError(err, "could not create a projectfile")

	err = fileutils.Touch(filepath.Join(suite.dir, "test-file"))
	suite.NoError(err, "could not create a temp file")

	_, err = worktree.Add("test-file")
	suite.NoError(err, "could not add tempfile to staging")

	_, err = worktree.Add("activestate.yaml")
	suite.NoError(err, "could not add projectfile to staging")

	commit, err := worktree.Commit("commit for test", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "testing",
			Email: "testing@testing.org",
			When:  time.Now(),
		},
	})
	suite.NoError(err, "could not create a commit")

	_, err = repo.CommitObject(commit)
	suite.NoError(err, "could not commit testfile")

	suite.anotherDir, err = os.MkdirTemp("", "TestMoveFiles")
	suite.NoError(err, "could not create another temporary directory")
}

func (suite *GitTestSuite) AfterTest(suiteName, testName string) {
	err := os.RemoveAll(suite.dir)
	if err != nil {
		fmt.Printf("WARNING: Could not remove temp dir: %s, error: %v", suite.dir, err)
	}
	err = os.RemoveAll(suite.anotherDir)
	if err != nil {
		fmt.Printf("WARNING: Could not remove temp dir: %s, error: %v", suite.dir, err)
	}
}

func (suite *GitTestSuite) TestEnsureCorrectProject() {
	err := runbitsGit.EnsureCorrectProject("test-owner", "test-project", filepath.Join(suite.dir, constants.ConfigFileName), "test-repo", outputhelper.NewCatcher(), blackhole.New())
	suite.NoError(err, "projectfile URL should contain owner and name")
}

func (suite *GitTestSuite) TestEnsureCorrectProject_Missmatch() {
	owner := "not-owner"
	name := "bad-project"
	projectPath := filepath.Join(suite.dir, constants.ConfigFileName)
	actualCatcher := outputhelper.NewCatcher()
	err := runbitsGit.EnsureCorrectProject(owner, name, projectPath, "test-repo", actualCatcher, blackhole.New())
	suite.NoError(err)

	proj, err := project.Parse(projectPath)
	suite.NoError(err)

	expectedCatcher := outputhelper.NewCatcher()
	expectedCatcher.Notice(locale.Tr("warning_git_project_mismatch", "test-repo", project.NewNamespace(owner, name, "").String(), constants.DocumentationURLMismatch))

	suite.Equal(expectedCatcher.CombinedOutput(), actualCatcher.CombinedOutput())
	suite.Equal(owner, proj.Owner())
	suite.Equal(name, proj.Name())
}

func (suite *GitTestSuite) TestMoveFiles() {
	anotherDir := filepath.Join(suite.anotherDir, "anotherDir")
	err := runbitsGit.MoveFiles(suite.dir, anotherDir)
	suite.NoError(err, "should be able to move files wihout error")

	_, err = os.Stat(filepath.Join(anotherDir, constants.ConfigFileName))
	suite.NoError(err, "file should be moved")

	_, err = os.Stat(filepath.Join(anotherDir, "test-file"))
	suite.NoError(err, "file should be moved")
}

func (suite *GitTestSuite) TestMoveFilesDirNoEmpty() {
	anotherDir := filepath.Join(suite.anotherDir, "anotherDir")
	err := os.MkdirAll(anotherDir, 0755)
	suite.NoError(err, "should be able to create another temp directory")

	err = fileutils.Touch(filepath.Join(anotherDir, "file.txt"))
	suite.Require().NoError(err)

	err = runbitsGit.MoveFiles(suite.dir, anotherDir)
	expected := locale.WrapError(err, "err_git_verify_dir", "Could not verify destination directory")
	suite.EqualError(err, expected.Error())
}

func TestGitTestSuite(t *testing.T) {
	suite.Run(t, new(GitTestSuite))
}
