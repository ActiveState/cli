package remove

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/hooks"
	"github.com/ActiveState/cli/pkg/cmdlets/variables"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
)

// Copies the activestate config file in the root test/ directory into the local
// config directory, reads the config file as a project, and returns that
// project.
func getTestProject(t *testing.T) *projectfile.Project {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Got root path")
	src := filepath.Join(root, "test", constants.ConfigFileName)
	dst := filepath.Join(config.GetDataDir(), constants.ConfigFileName)
	fail := fileutils.CopyFile(src, dst)
	assert.Nil(t, fail, "Copied test activestate config file")
	project, err := projectfile.Parse(dst)
	assert.NoError(t, err, "Parsed test config file")
	return project
}

func setup(t *testing.T) {
	Args.Identifier = ""
	testPromptResultOverride = ""
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})
	projectfile.Reset()
}

func TestExecute(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	assert := assert.New(t)
	Command.Execute()
	assert.Equal(true, true, "Execute didn't panic")
}

func TestRemoveByHashCmd(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	cmdName := "REMOVE_ME"

	hook := projectfile.Hook{Name: cmdName, Value: "This is a command"}
	project.Hooks = append(project.Hooks, hook)
	project.Save()

	hash, _ := hook.Hash()
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{hash})
	Command.Execute()
	Cc.SetArgs([]string{})

	project = projectfile.Get()
	mappedHooks, _ := hooks.HashHooksFiltered(project.Hooks, []string{cmdName})
	assert.Equal(t, 0, len(mappedHooks), fmt.Sprintf("No hooks should be found of name: '%v'", cmdName))
}

func TestRemoveByNameCmd(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	cmdName := "REMOVE_ME"

	hook := projectfile.Hook{Name: cmdName, Value: "This is a command"}
	project.Hooks = append(project.Hooks, hook)
	project.Save()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{cmdName})
	Command.Execute()
	Cc.SetArgs([]string{})

	project = projectfile.Get()
	mappedHooks, _ := hooks.HashHooksFiltered(project.Hooks, []string{cmdName})
	assert.Equal(t, 0, len(mappedHooks), fmt.Sprintf("No hooks should be found of name: '%v', found: %v", cmdName, mappedHooks))
}

func TestRemovePrompt(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	options, optionsMap, err := hooks.PromptOptions("FIRST_INSTALL")
	print.Formatted("\nmap1: %v\n", optionsMap)
	assert.NoError(t, err, "Should be able to get prompt options")

	testPromptResultOverride = options[0]

	removed := removeByPrompt("FIRST_INSTALL")
	assert.NotNil(t, removed, "Received a removed hook")

	hash, _ := removed.Hash()
	assert.Equal(t, optionsMap[testPromptResultOverride], hash, "Should have removed one hook")
}

func TestRemoveByHash(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	hookLen := len(project.Hooks)

	hash, err := project.Hooks[0].Hash()
	assert.NoError(t, err, "Should get hash")
	removed := removeByHash(hash)
	assert.NotNil(t, removed, "Received a removed hook")

	project = projectfile.Get()
	assert.Equal(t, hookLen-1, len(project.Hooks), "One hook should have been removed")
}

func TestRemovebyName(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	hookLen := len(project.Hooks)

	removed := removeByName(project.Hooks[0].Name)
	assert.NotNil(t, removed, "Received a removed hook")

	assert.Equal(t, hookLen-1, len(project.Hooks), "One hook should have been removed")
}

// This test shoudln't remove anything as there are multiple hooks configured for the same hook name
func TestRemoveByNameFailCmd(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	cmdName := "REMOVE_ME"

	hook1 := projectfile.Hook{Name: cmdName, Value: "This is a command"}
	hook2 := projectfile.Hook{Name: cmdName, Value: "This is another command"}
	project.Hooks = append(project.Hooks, hook1)
	project.Hooks = append(project.Hooks, hook2)
	project.Save()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{cmdName})
	Command.Execute()
	Cc.SetArgs([]string{})

	mappedHooks, _ := hooks.HashHooksFiltered(project.Hooks, []string{cmdName})
	assert.Equal(t, 2, len(mappedHooks), fmt.Sprintf("There should still be two commands of the same name in the config: '%v'", cmdName))
}

func TestRemoveNonExistant(t *testing.T) {
	setup(t)
	project := getTestProject(t)
	project.Persist()

	_, _, err := variables.PromptOptions("")
	assert.NoError(t, err, "Should be able to get prompt options")

	testPromptResultOverride = "does-not-exist"

	removed := removeByPrompt("")
	assert.Nil(t, removed, "Could not remove non-existant hook")
}
