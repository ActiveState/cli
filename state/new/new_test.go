package new

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/stretchr/testify/assert"
)

func TestNewInEmptyDir(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "cli-new-test")
	assert.NoError(t, err, "Created temp directory")
	cwd, err := os.Getwd()
	assert.NoError(t, err, "Fetched cwd")

	err = os.Chdir(tmpdir)
	assert.NoError(t, err, "Switched to tempdir")
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-o", "test-owner", "-v", "1.0"})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	_, err = os.Stat(filepath.Join(tmpdir, constants.ConfigFileName))
	assert.NoError(t, err, "Project was created")
	err = os.Rename(constants.ConfigFileName, constants.ConfigFileName+".bak")
	assert.NoError(t, err, "Renamed config file so later tests cannot reference it")
	os.Chdir(cwd) // restore

	os.RemoveAll(tmpdir)
}

func TestNewInNonEmptyDir(t *testing.T) {
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	tmpdir, _ := ioutil.TempDir("", "cli-new-test")
	cwd, _ := os.Getwd()
	err := ioutil.WriteFile(filepath.Join(tmpdir, "foo.txt"), []byte(""), 0666)
	assert.NoError(t, err, "Wrote dummy file")

	os.Chdir(tmpdir)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-o", "test-owner", "-v", "1.0"})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	_, err = os.Stat(filepath.Join(tmpdir, "test-name", constants.ConfigFileName))
	assert.NoError(t, err, "Project was created in sub-directory")
	os.Chdir(cwd) // restore

	os.RemoveAll(tmpdir)
}

func TestNewWithPathToExistingDir(t *testing.T) {
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	tmpdir, _ := ioutil.TempDir("", "cli-new-test")
	cwd, _ := os.Getwd()
	err := ioutil.WriteFile(filepath.Join(tmpdir, "foo.txt"), []byte(""), 0666)
	assert.NoError(t, err, "Wrote dummy file")

	os.Chdir(tmpdir)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-p", tmpdir, "-o", "test-owner", "-v", "1.0"})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	_, err = os.Stat(filepath.Join(tmpdir, constants.ConfigFileName))
	assert.Error(t, err, "Project was not created in existing directory")
	os.Chdir(cwd) // restore

	os.RemoveAll(tmpdir)
}

func TestNewWithBadPath(t *testing.T) {
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	invalidPath := "/invalid-path:"
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-p", invalidPath, "-o", "test-owner", "-v", "1.0"})
	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	_, err = os.Stat(invalidPath)
	assert.Error(t, err, "Project was not created")
}

func TestNewWithBadVersion(t *testing.T) {
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	tmpdir, _ := ioutil.TempDir("", "cli-new-test")
	cwd, _ := os.Getwd()

	os.Chdir(tmpdir)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-o", "test-owner", "-v", "badVersion"})
	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	_, err = os.Stat(filepath.Join(tmpdir, constants.ConfigFileName))
	assert.Error(t, err, "Project was not created")
	os.Chdir(cwd) // restore

	os.RemoveAll(tmpdir)
}
