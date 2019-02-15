package run

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestRunStandaloneCommand(t *testing.T) {
	Args.Name = "" // reset
	failures.ResetHandled()

	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = strings.TrimSpace(`
scripts:
  - name: run
    value: echo foo
    standalone: true
  `)
	} else {
		contents = strings.TrimSpace(`
scripts:
  - name: run
    value: cmd /C echo foo
    standalone: true
  `)
	}
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"run"})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")
}

func TestRunMissingCommandName(t *testing.T) {
	Args.Name = "" // reset
	failures.ResetHandled()

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
scripts:
  - name: run
    value: whatever
  `)
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{""})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")

	handled := failures.Handled()
	require.NotNil(t, handled, "expected a failure")
	assert.Equal(t, failures.FailUserInput, handled.(*failures.Failure).Type, "No failure occurred")
}

func TestRunUnknownCommandName(t *testing.T) {
	Args.Name = "" // reset
	failures.ResetHandled()

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
scripts:
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
	Args.Name = "" // reset
	failures.ResetHandled()

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
scripts:
  - name: run
    value: whatever
    standalone: true
  `)
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Command.Register()
	exitCode := 0
	Command.Exiter = func(code int) {
		exitCode = code
	}

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"run"})
	err = Command.Execute()
	assert.Equal(t, 127, exitCode, "Execution caused exit")
	assert.Error(t, failures.Handled(), "Failure occurred")
}

func TestRunActivatedCommand(t *testing.T) {
	Args.Name = "" // reset
	failures.ResetHandled()

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
scripts:
  - name: run
    value: echo foo`)
	} else {
		contents = strings.TrimSpace(`
scripts:
  - name: run
    value: cmd /C echo foo`)
	}
	err = yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	// Run the command.
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"run"})
	failures.ResetHandled()
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
	assert.NoError(t, failures.Handled(), "No failure occurred")

	// Reset.
	projectfile.Reset()
}
