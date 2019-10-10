package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
)

type GitTestSuite struct {
	suite.Suite
	dir        string
	anotherDir string
}

func (suite *GitTestSuite) BeforeTest(suiteName, testName string) {
	var err error
	suite.dir, err = ioutil.TempDir("", testName)
	suite.NoError(err, "should be able to create a temporary directory")

	projectURL := fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, "test-owner", "test-project")

	_, fail := projectfile.Create(projectURL, suite.dir)
	suite.NoError(fail.ToError(), "should be able to create a projectfile")

	tempFile, fail := fileutils.Touch(filepath.Join(suite.dir, "test-file"))
	suite.NoError(fail.ToError(), "should be able to create a temp file")
	defer tempFile.Close()

	suite.anotherDir, err = ioutil.TempDir("", "TestMoveFiles")
	suite.NoError(err, "should be able to create another temp directory")
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

func (suite *GitTestSuite) TestCloneProjectRepo() {
	fail := CloneProjectRepo("test-owner", "test-name", "does-not-matter")
	suite.NoError(fail.ToError(), "should not get error")
}

func (suite *GitTestSuite) TestEnsureCorrectRepo() {
	fail := ensureCorrectRepo("test-owner", "test-project", filepath.Join(suite.dir, constants.ConfigFileName))
	suite.NoError(fail.ToError(), "projectfile URL should contain owner and name")
}

func (suite *GitTestSuite) TestEnsureCorrectRepo_Mistmatch() {
	fail := ensureCorrectRepo("not-owner", "bad-project", filepath.Join(suite.dir, constants.ConfigFileName))
	expected := FailProjectURLMismatch.New(locale.T("error_git_project_url_mismatch"))
	suite.EqualError(fail, expected.Error(), "expected errors to match")
}

func (suite *GitTestSuite) TestMoveFiles() {
	anotherDir := filepath.Join(suite.anotherDir, "anotherDir")
	fail := moveFiles(suite.dir, anotherDir)
	suite.NoError(fail.ToError(), "should be able to move files wihout error")

	_, err := os.Stat(filepath.Join(anotherDir, constants.ConfigFileName))
	suite.NoError(err, "file should be moved")

	_, err = os.Stat(filepath.Join(anotherDir, "test-file"))
	suite.NoError(err, "file should be moved")
}

func (suite *GitTestSuite) TestMoveFilesDirInUse() {
	anotherDir := filepath.Join(suite.anotherDir, "anotherDir")
	err := os.MkdirAll(anotherDir, 0755)
	suite.NoError(err, "should be able to create another temp directory")

	fail := moveFiles(suite.dir, anotherDir)
	expected := FailTargetDirInUse.New(locale.T("error_git_target_dir_exists"))
	suite.EqualError(fail, expected.Error())
}

func TestGitTestSuite(t *testing.T) {
	suite.Run(t, new(GitTestSuite))
}
