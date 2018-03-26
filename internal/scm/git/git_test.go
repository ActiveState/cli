package git

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
)

func TestMatches(t *testing.T) {
	assert.True(t, MatchesRemote("git@github.com:golang/playground.git"), "This is a Git repo")
	assert.True(t, MatchesRemote("http://github.com/golang/playground"), "This is a Git repo")
	assert.True(t, MatchesRemote("https://github.com/golang/playground"), "This is a Git repo")
	assert.False(t, MatchesRemote("nttp://github.com/golang/playground"), "This invalid Github URL is not a Git repo")
	assert.False(t, MatchesRemote("http://github.com/golang"), "This invalid Github URL is not a Git repo")

	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	assert.True(t, MatchesRemote(filepath.Join(root, "internal", "scm", "git", "testdata", "repo")), "This is a git remote")
	assert.False(t, MatchesPath(filepath.Join(root, "internal", "scm", "git", "testdata", "repo")), "This shouldnt match as this is a remote, not a checkout")
	assert.True(t, MatchesPath(root), "This is a git repo")

	// TODO: include testdata from future SCMs.
	assert.False(t, MatchesRemote("http://www.selenic.com/hg"))
	assert.False(t, MatchesRemote("file:///var/svn/repos/test"))
}

func TestHumanishPart(t *testing.T) {
	assert.Equal(t, "playground", NewFromURI("git@github.com:golang/playground.git").humanishPart(), "Got the expected humanish part")
	assert.Equal(t, "playground", NewFromURI("http://github.com/golang/playground").humanishPart(), "Got the expected humanish part")
	assert.Equal(t, "playground", NewFromURI("https://github.com/golang/playground").humanishPart(), "Got the expected humanish part")

	// From `git help clone` documentation.
	assert.Equal(t, "repo", NewFromURI("/path/to/repo.git").humanishPart(), "Got the expected humanish part")
	assert.Equal(t, "foo", NewFromURI("host.xz:foo/.git").humanishPart(), "Got the expected humanish part")
}

func TestClone(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	err = os.Chdir(filepath.Join(root, "test"))
	assert.NoError(t, err, "Moving to new CWD")

	cwd, err := os.Getwd() // store
	assert.NoError(t, err, "Saving CWD")
	repo, err := filepath.Abs(filepath.Join(root, "internal", "scm", "git", "testdata", "repo"))
	assert.Nil(t, err, "The test repository exists")

	tempdir, err := ioutil.TempDir("", "cli-")
	assert.Nil(t, err, "A temporary directory was created")
	err = os.Chdir(tempdir)
	assert.Nil(t, err, "Changed into temporary directory")

	// Test basic clone.
	_, err = os.Stat("repo")
	assert.True(t, os.IsNotExist(err), "The cloned repository does not exist yet")
	git := NewFromURI(repo)
	err = git.Clone()
	assert.Nil(t, err, "The remote repository exists")
	assert.Equal(t, filepath.Base(git.Path()), "repo", "The repository was cloned into the expected directory")
	_, err = os.Stat(git.Path())
	assert.Nil(t, err, "The cloned repository exists")
	files := []string{"foo.txt", "bar.txt", "baz.txt"}
	for _, file := range files {
		_, err = os.Stat(filepath.Join(git.Path(), file))
		assert.Nil(t, err, "The cloned repository contains an expected file")
	}

	// Test clone with specified directory.
	_, err = os.Stat("repo2")
	assert.True(t, os.IsNotExist(err), "The cloned repository does not exist yet")
	git = NewFromURI(repo)
	git.SetPath(filepath.Join(tempdir, "repo2"))
	err = git.Clone()
	assert.Nil(t, err, "The remote repository exists")
	assert.Equal(t, filepath.Base(git.Path()), "repo2", "The repository was cloned into the expected directory")
	_, err = os.Stat(git.Path())
	assert.Nil(t, err, "The cloned repository exists")
	for _, file := range files {
		_, err = os.Stat(filepath.Join(git.Path(), file))
		assert.Nil(t, err, "The cloned repository contains an expected file")
	}

	// Test a non-existant repo.
	git = NewFromURI("does-not-exist")
	err = git.Clone()
	assert.Error(t, err, "The repository could not be cloned")
	assert.NotEqual(t, git.Path(), "", "The repository would have been cloned into a directory")
	_, err = os.Stat(git.Path())
	assert.True(t, os.IsNotExist(err), "The non-existant repository was not cloned")

	err = os.Chdir(cwd) // restore
	assert.Nil(t, err, "Changed back to original directory")
	err = os.RemoveAll(tempdir) // clean up
	assert.Nil(t, err, "The temporary directory was removed")
}

func TestRepoExists(t *testing.T) {
	originalCWD, err := os.Getwd()
	assert.NoError(t, err, "Saving CWD")

	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	newCWD, err := filepath.Abs(filepath.Join(root, "internal", "scm", "git", "testdata"))
	assert.NoError(t, err, "Should detect root path")
	err = os.Chdir(newCWD)
	assert.NoError(t, err, "Moving to new CWD")

	repoExists, err := filepath.Abs(filepath.Join(root, "internal", "scm", "git", "testdata", "repo"))
	assert.Nil(t, err, "Obtain repo directory")
	gitExists := NewFromURI(repoExists)
	assert.True(t, gitExists.TargetExists(), "Repo should already exist")

	repoFake, err := filepath.Abs(filepath.Join(root, "internal", "scm", "git", "testdata", "fakerepo"))
	assert.Nil(t, err, "Obtain repo directory")
	gitFake := NewFromURI(repoFake)
	assert.False(t, gitFake.TargetExists(), "Repo should not exist")

	err = os.Chdir(originalCWD)
	assert.NoError(t, err, "Moving back to original CWD")
}

func TestDetectUri(t *testing.T) {
	path, err := environment.GetRootPath()
	assert.NoError(t, err, "Retrieved root path")

	fmt.Println(path)

	scm := NewFromPath(path)
	assert.NotEmpty(t, scm.URI(), "Can detect remote")
}
