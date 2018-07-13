package run

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestRunStandalone(t *testing.T) {
	Flags.Standalone, Args.Name = false, "" // reset
	os.Setenv("SHELL", "bash")

	tmpfile, err := ioutil.TempFile("", "testRunCommand")
	assert.NoError(t, err)
	tmpfile.Close()
	os.Remove(tmpfile.Name())

	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = fmt.Sprintf(`
commands:
  - name: run
    value: |
      echo "Hello"
      touch %s`, tmpfile.Name())
	} else {
		contents = fmt.Sprintf(`
commands:
  - name: run
    value: |
    echo "Hello"
    copy NUL %s`, tmpfile.Name())
	}
	err = yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--standalone"})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
	print.Line(tmpfile.Name())
	assert.FileExists(t, tmpfile.Name())
}

func TestRunStandaloneCommand(t *testing.T) {
	Flags.Standalone, Args.Name = false, "" // reset

	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = strings.TrimSpace(`
commands:
  - name: run
    value: echo foo
    standalone: true
    `)
	} else {
		contents = strings.TrimSpace(`
commands:
  - name: run
    value: cmd /C echo foo
    standalone: true
    `)
	}
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{""})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestRunUnknownCommandName(t *testing.T) {
	Flags.Standalone, Args.Name = false, "" // reset

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
commands:
  - name: run
    value: whatever
  `)
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"unknown"})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestRunUnknownCommand(t *testing.T) {
	Flags.Standalone, Args.Name = false, "" // reset

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
commands:
  - name: run
    value: whatever
  `)
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--standalone"})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.Error(t, failures.Handled(), "Failure occurred")
}

func TestRunActivatedCommand(t *testing.T) {
	Flags.Standalone, Args.Name = false, "" // reset

	// Prepare an empty activated environment.
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
	datadir := config.GetDataDir()
	os.RemoveAll(filepath.Join(datadir, "virtual"))
	os.RemoveAll(filepath.Join(datadir, "packages"))
	os.RemoveAll(filepath.Join(datadir, "languages"))
	os.RemoveAll(filepath.Join(datadir, "artifacts"))

	// Setup the project.
	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = strings.TrimSpace(`
commands:
  - name: run
    value: echo foo`)
	} else {
		contents = strings.TrimSpace(`
commands:
  - name: run
    value: cmd /C echo foo`)
	}
	err = yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	// Run the command.
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	// Reset.
	projectfile.Reset()
}
