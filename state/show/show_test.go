package show

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/assert"
)

func TestShow(t *testing.T) {
	Args.Remote = "" // reset
	Flags.Output = new(string)
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	src := filepath.Join(root, "test", constants.ConfigFileName)
	dst := filepath.Join(root, "state", "show", "testdata", "generated", "config", constants.ConfigFileName)
	fail := fileutils.CopyFile(src, dst)
	assert.Nil(t, fail, "Copied test activestate config file")

	cwd, err := os.Getwd()
	assert.NoError(t, err, "Fetched cwd")
	err = os.Chdir(filepath.Join(root, "state", "show", "testdata", "generated", "config"))
	assert.NoError(t, err, "Changed into generated config dir")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{""})
	Cc.Execute()
	assert.NoError(t, err, "Executed without error")

	os.Chdir(cwd) // restore
}

func TestShowLocal(t *testing.T) {
	Args.Remote = "" // reset
	Flags.Output = new(string)
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{filepath.Join(root, "test")})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestShowFailDirDoesNotExist(t *testing.T) {
	Args.Remote = "" // reset
	Flags.Output = new(string)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"/:does-not-exist"})
	err := Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestShowFailNoConfigFile(t *testing.T) {
	Args.Remote = "" // reset
	Flags.Output = new(string)
	tmpdir, err := ioutil.TempDir("", "cli-show-test")
	assert.NoError(t, err, "Created temp directory")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{tmpdir})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	os.RemoveAll(tmpdir)
}

func TestShowFailParseConfig(t *testing.T) {
	Args.Remote = "" // reset
	Flags.Output = new(string)
	tmpdir, err := ioutil.TempDir("", "cli-show-test")
	assert.NoError(t, err, "Created temp directory")

	err = ioutil.WriteFile(filepath.Join(tmpdir, constants.ConfigFileName), []byte("\tBad Syntax"), 0666)
	assert.NoError(t, err, "Created bad configuration file")

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{tmpdir})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	os.RemoveAll(tmpdir)
}
