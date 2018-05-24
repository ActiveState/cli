package new

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "cli-new-test")
	assert.NoError(t, err, "Created temp directory")
	cwd, err := os.Getwd()
	assert.NoError(t, err, "Fetched cwd")

	// Test creating project in empty directory.
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

	// Test creating project in non-empty directory.
	os.Chdir(tmpdir)
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	Cc.SetArgs([]string{"test-name", "-o", "test-owner", "-v", "1.0"})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	_, err = os.Stat(filepath.Join(tmpdir, "test-name", constants.ConfigFileName))
	assert.NoError(t, err, "Project was created in sub-directory")
	os.Chdir(cwd) // restore

	// Test creating project in existing directory.
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	Cc.SetArgs([]string{"test-name", "-p", tmpdir, "-o", "test-owner", "-v", "1.0"})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	_, err = os.Stat(filepath.Join(tmpdir, constants.ConfigFileName))
	assert.Error(t, err, "Project was not created in existing directory")

	os.RemoveAll(tmpdir)
}

func TestNewBadPath(t *testing.T) {
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	invalidPath := ":invalid:path:"
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-p", invalidPath, "-o", "test-owner", "-v", "1.0"})
	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	_, err = os.Stat(filepath.Join(invalidPath, constants.ConfigFileName))
	assert.Error(t, err, "Project was not created")
}

func TestNewBadVersion(t *testing.T) {
	Flags.Path, Flags.Owner, Flags.Version, Args.Name = "", "", "", "" // reset
	tmpdir, err := ioutil.TempDir("", "cli-new-test")
	assert.NoError(t, err, "Created temp directory")
	cwd, err := os.Getwd()
	assert.NoError(t, err, "Fetched cwd")
	os.Chdir(tmpdir)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"test-name", "-o", "test-owner", "-v", "badVersion"})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	_, err = os.Stat(filepath.Join(tmpdir, constants.ConfigFileName))
	assert.Error(t, err, "Project was not created")
	os.Chdir(cwd) // restore
	os.RemoveAll(tmpdir)
}
