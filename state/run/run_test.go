package run

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestRunStandalone(t *testing.T) {
	Flags.Standalone, Args.Name = false, "" // reset

	project := &projectfile.Project{}
	var contents string
	if runtime.GOOS != "windows" {
		contents = strings.TrimSpace(`
commands:
  - name: run
    value: echo foo
    `)
	} else {
		contents = strings.TrimSpace(`
commands:
  - name: run
    value: cmd /C echo foo
    `)
	}
	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--standalone"})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")
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
    value: echo foo
    `)
	} else {
		contents = strings.TrimSpace(`
commands:
  - name: run
    value: cmd /C echo foo
    `)
	}
	err = yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	// Run the command.
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})
	err = Command.Execute()
	assert.NoError(t, err, "Executed without error")

	// Reset.
	projectfile.Reset()
}
