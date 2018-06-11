package add

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/cmdlets/hooks"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
)

// Copies the activestate config file in the root test/ directory into the local
// config directory, reads the config file as a project, and returns that
// project.
func getTestProject(t *testing.T) *projectfile.Project {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Got root path")
	src := filepath.Join(root, "test", constants.ConfigFileName)
	dst := filepath.Join(root, "state", "hook", "add", "testdata", "generated", "config", constants.ConfigFileName)
	fail := fileutils.CopyFile(src, dst)
	assert.Nil(t, fail, "Copied test activestate config file")
	project, err := projectfile.Parse(dst)
	assert.NoError(t, err, "Parsed test config file")
	return project
}

func TestAddHookPass(t *testing.T) {
	Args.Hook, Args.Command = "", "" // reset
	project := getTestProject(t)
	project.Persist()

	newHookName := "ACTIVATE"
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{newHookName, "echo 'This is a command'"})
	Cc.Execute()

	var found = false
	for _, hook := range project.Hooks {
		if hook.Name == newHookName {
			found = true
		}
	}
	assert.True(t, found, fmt.Sprintf("Should find a hook named %v", newHookName))
}

func TestAddHookFail(t *testing.T) {
	Args.Hook, Args.Command = "", "" // reset
	project := getTestProject(t)
	project.Persist()

	Cc := Command.GetCobraCmd()
	newHookName := "A_HOOK"
	Cc.SetArgs([]string{newHookName})
	Cc.Execute()

	var found = false
	for _, hook := range project.Hooks {
		if hook.Name == newHookName {
			found = true
		}
	}
	assert.False(t, found, fmt.Sprintf("Should NOT find a hook named %v", newHookName))
}

// Test it doesn't explode when run with no args
func TestExecute(t *testing.T) {
	Args.Hook, Args.Command = "", "" // reset
	project := getTestProject(t)
	project.Persist()

	Command.Execute()

	assert.Equal(t, true, true, "Execute didn't panic")
}

//
func TestAddHookFailIdentical(t *testing.T) {
	Args.Hook, Args.Command = "", "" // reset
	project := getTestProject(t)
	project.Persist()

	hookName := "ACTIVATE"
	value := "echo 'This is a command'"
	hook1 := projectfile.Hook{Name: hookName, Value: value}
	project.Hooks = append(project.Hooks, hook1)
	project.Save()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{hookName, value})
	Cc.Execute()

	filteredMappedHooks, _ := hooks.HashHooksFiltered(project.Hooks, []string{hookName})

	assert.Equal(t, 1,
		len(filteredMappedHooks),
		fmt.Sprintf("There should be only one hook configure for hookname'%v'", hookName))
}
