package git

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/pkg/platform/api"
	authMock "github.com/ActiveState/cli/pkg/platform/authentication/mock"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type GitTestSuite struct {
	suite.Suite
	authMock   *authMock.Mock
	graphMock  *httpmock.HTTPMock
	dir        string
	anotherDir string
}

func (suite *GitTestSuite) BeforeTest(suiteName, testName string) {
	suite.authMock = authMock.Init()
	suite.graphMock = httpmock.Activate(api.GetServiceURL(api.ServiceGraphQL).String())

	var err error
	suite.dir, err = ioutil.TempDir("", testName)
	suite.NoError(err, "could not create a temporary directory")

	repo, err := git.PlainInit(suite.dir, false)
	suite.NoError(err, "could not init a new git repo")

	worktree, err := repo.Worktree()
	suite.NoError(err, "could not get repository worktree")

	projectURL := fmt.Sprintf("https://%s/%s/%s", constants.PlatformURL, "test-owner", "test-project")

	_, fail := projectfile.Create(projectURL, suite.dir)
	suite.NoError(fail.ToError(), "could not create a projectfile")

	tempFile, fail := fileutils.Touch(filepath.Join(suite.dir, "test-file"))
	suite.NoError(fail.ToError(), "could not create a temp file")

	err = tempFile.Close()
	suite.NoError(err, "could not close file")

	_, err = worktree.Add("test-file")
	suite.NoError(err, "could not add tempfile to staging")

	_, err = worktree.Add("activestate.yaml")

	commit, err := worktree.Commit("commit for test", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "testing",
			Email: "testing@testing.org",
			When:  time.Now(),
		},
	})

	_, err = repo.CommitObject(commit)
	suite.NoError(err, "could not commit testfile")

	suite.anotherDir, err = ioutil.TempDir("", "TestMoveFiles")
	suite.NoError(err, "could not create another temporary directory")
}

func (suite *GitTestSuite) AfterTest(suiteName, testName string) {
	suite.authMock.Close()
	httpmock.DeActivate()

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
	type tempProject struct {
		Name           string `json:"name"`
		RepoURL        string `json:"repo_url"`
		OrganizationID string `json:"organization_id"`
	}

	suite.authMock.MockLoggedin()

	response := `{"data": {"projects": [%s]}}`
	proj := tempProject{
		Name:           "clone",
		RepoURL:        suite.dir + "/.git",
		OrganizationID: "00010001-0001-0001-0001-000100010001",
	}

	file, err := json.MarshalIndent(proj, "", " ")
	suite.NoError(err, "could not marshall tempProject struct")

	data := fmt.Sprintf(response, string(file))
	suite.graphMock.RegisterWithResponseBody("POST", "", 200, string(data))

	targetDir := filepath.Join(suite.dir, "target-clone-dir")

	repo := NewRepo()
	fail := repo.CloneProject("test-owner", "test-project", targetDir)
	suite.Require().NoError(fail.ToError(), "should clone without issue")
	suite.FileExists(filepath.Join(targetDir, "activestate.yaml"), "activestate.yaml file should have been cloned")
	suite.FileExists(filepath.Join(targetDir, "test-file"), "tempororary file should have been cloned")
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

func TestGitTestSuite(t *testing.T) {
	suite.Run(t, new(GitTestSuite))
}
