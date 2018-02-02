package activate

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"activate"})

	Command.Execute()

	assert.Equal(true, true, "Execute didn't panic")
}

func TestExecuteGitClone(t *testing.T) {
	cwd, _ := os.Getwd() // store
	repo, err := filepath.Abs(filepath.Join("..", "..", "internal", "scm", "git", "testdata", "repo"))
	assert.Nil(t, err, "The test Git repository exists")

	tempdir, err := ioutil.TempDir("", "ActiveState-CLI-")
	assert.Nil(t, err, "A temporary directory was created")
	err = os.Chdir(tempdir)
	assert.Nil(t, err, "Changed into temporary directory")

	// Test basic clone.
	_, err = os.Stat("repo")
	assert.True(t, os.IsNotExist(err), "The cloned repository does not exist yet")
	Command.GetCobraCmd().SetArgs([]string{repo})
	Command.Execute()
	_, err = os.Stat("repo")
	assert.Nil(t, err, "The cloned repository exists")
	files := []string{"foo.txt", "bar.txt", "baz.txt"}
	for _, file := range files {
		_, err = os.Stat(filepath.Join("repo", file))
		assert.Nil(t, err, "The cloned repository contains an expected file")
	}

	// Test clone with specified directory.
	_, err = os.Stat("repo2")
	assert.True(t, os.IsNotExist(err), "The cloned repository does not exist yet")
	Command.GetCobraCmd().SetArgs([]string{repo, "--path", "repo2"})
	Command.Execute()
	_, err = os.Stat("repo2")
	assert.Nil(t, err, "The cloned repository exists")
	for _, file := range files {
		_, err = os.Stat(filepath.Join("repo2", file))
		assert.Nil(t, err, "The cloned repository contains an expected file")
	}

	// Test clone with specified directory and cd.
	_, err = os.Stat("repo3")
	assert.True(t, os.IsNotExist(err), "The cloned repository does not exist yet")
	Command.GetCobraCmd().SetArgs([]string{repo, "--path", "repo3", "--cd"})
	Command.Execute()
	newCwd, _ := os.Getwd()
	assert.Equal(t, "repo3", filepath.Base(newCwd), "The cloned repository exists and was changed into")
	for _, file := range files {
		_, err = os.Stat(file)
		assert.Nil(t, err, "The cloned repository contains an expected file")
	}

	err = os.Chdir(cwd) // restore
	assert.Nil(t, err, "Changed back to original directory")
	err = os.RemoveAll(tempdir) // clean up
	assert.Nil(t, err, "The temporary directory was removed")
}
