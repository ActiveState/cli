package git

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/stretchr/testify/assert"
)

func TestIsGitURI(t *testing.T) {
	assert.True(t, IsGitURI("git@github.com:golang/playground.git"), "This is a Git repo")
	assert.True(t, IsGitURI("http://github.com/golang/playground"), "This is a Git repo")
	assert.True(t, IsGitURI("https://github.com/golang/playground"), "This is a Git repo")
	assert.False(t, IsGitURI("nttp://github.com/golang/playground"), "This invalid Github URL is not a Git repo")
	assert.False(t, IsGitURI("http://github.com/golang"), "This invalid Github URL is not a Git repo")

	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	assert.True(t, IsGitURI(filepath.Join(root, "internal", "scm", "git", "testdata", "repo")), "This is a git repo")

	// TODO: include testdata from future SCMs.
	assert.False(t, IsGitURI("http://www.selenic.com/hg"))
	assert.False(t, IsGitURI("file:///var/svn/repos/test"))
}

func TestHumanishPart(t *testing.T) {
	assert.Equal(t, "playground", (&Git{URI: "git@github.com:golang/playground.git"}).humanishPart(), "Got the expected humanish part")
	assert.Equal(t, "playground", (&Git{URI: "http://github.com/golang/playground"}).humanishPart(), "Got the expected humanish part")
	assert.Equal(t, "playground", (&Git{URI: "https://github.com/golang/playground"}).humanishPart(), "Got the expected humanish part")

	// From `git help clone` documentation.
	assert.Equal(t, "repo", (&Git{URI: "/path/to/repo.git"}).humanishPart(), "Got the expected humanish part")
	assert.Equal(t, "foo", (&Git{URI: "host.xz:foo/.git"}).humanishPart(), "Got the expected humanish part")
}

func TestConfigFileExists(t *testing.T) {
	if !WithinGithubRateLimit(2) {
		print.Warning("Exceeded Github API rate limit; skipping test 'TestConfigFileExists'")
		return // this test needs to call the Github API twice
	}

	git := &Git{URI: "https://github.com/ActiveState/repo"}
	assert.True(t, git.ConfigFileExists(), "The remote test repository has an ActiveState-CLI config file")

	git = &Git{URI: "https://github.com/ActiveState/does-not-exist"}
	assert.False(t, git.ConfigFileExists(), "The non-existant repository does not have an ActiveState-CLI config file")
}

func TestClone(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	cwd, _ := os.Getwd() // store
	repo, err := filepath.Abs(filepath.Join(root, "internal", "scm", "git", "testdata", "repo"))
	assert.Nil(t, err, "The test repository exists")

	tempdir, err := ioutil.TempDir("", "ActiveState-CLI-")
	assert.Nil(t, err, "A temporary directory was created")
	err = os.Chdir(tempdir)
	assert.Nil(t, err, "Changed into temporary directory")

	// Test basic clone.
	_, err = os.Stat("repo")
	assert.True(t, os.IsNotExist(err), "The cloned repository does not exist yet")
	git := &Git{URI: repo}
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
	git = &Git{URI: repo}
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
	git = &Git{URI: "does-not-exist"}
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
