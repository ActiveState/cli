package remove

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	hookhelper "github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/hooks"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/stretchr/testify/assert"
)

// For moving the CWD when needed during a test.
var startingDir string
var tempDir string

// Moves process into a tmp dir and brings a copy of project file with it
func moveToTmpDir() error {
	var err error
	startingDir, _ = os.Getwd()
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
	assert := assert.New(t)
	Command.Execute()
	assert.Equal(true, true, "Execute didn't panic")
}

func TestRemoveByHash(t *testing.T) {
	root, err := environment.GetRootPath()
	testDir := filepath.Join(root, "test")
	os.Chdir(testDir)
	assert.NoError(t, err, "Should detect root path")
	err = moveToTmpDir()
	assert.Nil(t, err, "A temporary directory was created and entered as CWD")

	project, _ := projectfile.Get()
	cmdName := "REMOVE_ME"

	hook := projectfile.Hook{Name: cmdName, Value: "This is a command"}
	project.Hooks = append(project.Hooks, hook)
	project.Save()

	hash, _ := hookhelper.HashHookStruct(hook)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{hash})
	Command.Execute()
	mappedHooks, _ := hookhelper.FilterHooks([]string{cmdName})
	assert.Equal(t, 0, len(mappedHooks), fmt.Sprintf("No hooks should be found of name: '%v'", cmdName))

	err = removeTmpDir()
	assert.Nil(t, err, "Tried to remove tmp testing dir")
}

func TestRemoveByName(t *testing.T) {
	root, err := environment.GetRootPath()
	testDir := filepath.Join(root, "test")
	os.Chdir(testDir)
	assert.NoError(t, err, "Should detect root path")
	err = moveToTmpDir()
	assert.Nil(t, err, "A temporary directory was created and entered as CWD")

	project, _ := projectfile.Get()
	cmdName := "REMOVE_ME"

	hook := projectfile.Hook{Name: cmdName, Value: "This is a command"}
	project.Hooks = append(project.Hooks, hook)
	project.Save()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{cmdName})
	Command.Execute()
	mappedHooks, _ := hookhelper.FilterHooks([]string{cmdName})
	assert.Equal(t, 0, len(mappedHooks), fmt.Sprintf("No hooks should be found of name: '%v'", cmdName))

	err = removeTmpDir()
	assert.Nil(t, err, "Tried to remove tmp testing dir")
}

// This test shoudln't remove anything as there are multiple hooks configured for the same hook name
func TestRemoveByNameFail(t *testing.T) {
	root, err := environment.GetRootPath()
	testDir := filepath.Join(root, "test")
	os.Chdir(testDir)
	assert.NoError(t, err, "Should detect root path")
	err = moveToTmpDir()
	assert.Nil(t, err, "A temporary directory was created and entered as CWD")
	cmdName := "REMOVE_ME"
	project, _ := projectfile.Get()

	hook1 := projectfile.Hook{Name: cmdName, Value: "This is a command"}
	hook2 := projectfile.Hook{Name: cmdName, Value: "This is another command"}
	project.Hooks = append(project.Hooks, hook1)
	project.Hooks = append(project.Hooks, hook2)
	project.Save()

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{cmdName})
	Command.Execute()
	mappedHooks, _ := hookhelper.FilterHooks([]string{cmdName})
	assert.Equal(t, 2, len(mappedHooks[cmdName]), fmt.Sprintf("There should still be two commands of the same name in the config: '%v'", cmdName))

	err = removeTmpDir()
	assert.Nil(t, err, "Tried to remove tmp testing dir")
}

// Setup projecet with two configured hooks of the same name.  Remove one.
func TestRemoveByIndex(t *testing.T) {
	root, err := environment.GetRootPath()
	testDir := filepath.Join(root, "test")
	os.Chdir(testDir)
	assert.NoError(t, err, "Should detect root path")
	err = moveToTmpDir()
	assert.Nil(t, err, "A temporary directory was created and entered as CWD")
	hookName := "REMOVE_ME"
	value1 := "echo cmd1"
	value2 := "echo cmd2"
	project, _ := projectfile.Get()

	hook1 := projectfile.Hook{Name: hookName, Value: value1}
	hook2 := projectfile.Hook{Name: hookName, Value: value2}
	project.Hooks = append(project.Hooks, hook1)
	project.Hooks = append(project.Hooks, hook2)
	project.Save()

	hooksLen := len(project.Hooks)
	filteredMappedHooks, _ := hookhelper.FilterHooks([]string{hookName})

	removeByIndex(1, filteredMappedHooks[hookName], project)
	assert.NotEqual(t, hooksLen, len(project.Hooks), "There should 3 hooks configured.")

	filteredMappedHooks, _ = hookhelper.FilterHooks([]string{hookName})

	assert.Equal(t, value2,
		filteredMappedHooks[hookName][0].Hook.Value,
		fmt.Sprintf("The first added hook should have been removed so remaining hook value should be '%v'", value2))

	err = removeTmpDir()
	assert.Nil(t, err, "Tried to remove tmp testing dir")

}
