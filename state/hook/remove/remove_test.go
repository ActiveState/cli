package remove

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/hooks"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/stretchr/testify/assert"
)

// For moving the CWD when needed during a test.
var startingDir string
var tempDir string

func setup(t *testing.T) {
	err := moveToTmpDir()
	assert.Nil(t, err, "A temporary directory was created and entered as CWD")

	Args.Identifier = ""
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})

	projectfile.Reset()
}

func teardown() {
	removeTmpDir()
}

// Moves process into a tmp dir and brings a copy of project file with it
func moveToTmpDir() error {
	var err error
	startingDir, _ = environment.GetRootPath()
	startingDir = filepath.Join(startingDir, "test")
	tempDir, err = ioutil.TempDir("", "ActiveSta bte-CLI-")
	if err != nil {
		return err
	}
	err = os.Chdir(tempDir)
	if err != nil {
		return err
	}

	copy(filepath.Join(startingDir, "activestate.yaml"),
		filepath.Join(tempDir, "activestate.yaml"))
	return nil
}

// Moves process to original dir and deletes temp
func removeTmpDir() error {
	err := os.Chdir(startingDir)
	if err != nil {
		return err
	}
	err = os.RemoveAll(tempDir)
	if err != nil {
		return err
	}
	return nil
}

func copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	in.Close()
	return out.Close()
}

func TestExecute(t *testing.T) {
	setup(t)
	defer teardown()

	assert := assert.New(t)
	Command.Execute()
	assert.Equal(true, true, "Execute didn't panic")
}

func TestRemoveByHashCmd(t *testing.T) {
	setup(t)
	defer teardown()

	project := projectfile.Get()
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
	defer teardown()

	project := projectfile.Get()
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
	defer teardown()

	options, optionsMap, err := hooks.PromptOptions("")
	print.Formatted("\nmap1: %v\n", optionsMap)
	assert.NoError(t, err, "Should be able to get prompt options")

	testPromptResultOverride = options[0]

	removed := removeByPrompt("")
	assert.NotNil(t, removed, "Received a removed hook")

	hash, _ := removed.Hash()
	assert.Equal(t, optionsMap[testPromptResultOverride], hash, "Should have removed one hook")
}

func TestRemoveByHash(t *testing.T) {
	setup(t)
	defer teardown()

	project := projectfile.Get()
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
	defer teardown()

	project := projectfile.Get()
	hookLen := len(project.Hooks)

	removed := removeByName(project.Hooks[0].Name)
	assert.NotNil(t, removed, "Received a removed hook")

	project = projectfile.Get()
	assert.Equal(t, hookLen-1, len(project.Hooks), "One hook should have been removed")
}

// This test shoudln't remove anything as there are multiple hooks configured for the same hook name
func TestRemoveByNameFailCmd(t *testing.T) {
	setup(t)
	defer teardown()

	cmdName := "REMOVE_ME"
	project := projectfile.Get()

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
