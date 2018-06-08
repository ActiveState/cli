package add

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/pkg/cmdlets/hooks"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
)

// For moving the CWD when needed during a test.
var startingDir string
var tempDir string

// Moves process into a tmp dir and brings a copy of project file with it
func moveToTmpDir() error {
	var err error
	root, err := environment.GetRootPath()
	testDir := filepath.Join(root, "test")
	os.Chdir(testDir)
	if err != nil {
		return err
	}
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

func TestAddHookPass(t *testing.T) {
	Args.Hook, Args.Command = "", "" // reset
	err := moveToTmpDir()

	assert.Nil(t, err, "A temporary directory was created and entered as CWD")

	newHookName := "ACTIVATE"
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{newHookName, "echo 'This is a command'"})
	Cc.Execute()

	project := projectfile.Get()
	var found = false
	for _, hook := range project.Hooks {
		if hook.Name == newHookName {
			found = true
		}
	}
	assert.True(t, found, fmt.Sprintf("Should find a hook named %v", newHookName))

	err = removeTmpDir()
	assert.Nil(t, err, "Tried to remove tmp testing dir")
}

func TestAddHookFail(t *testing.T) {
	Args.Hook, Args.Command = "", "" // reset
	err := moveToTmpDir()
	assert.Nil(t, err, "A temporary directory was created and entered as CWD")

	Cc := Command.GetCobraCmd()
	newHookName := "A_HOOK"
	Cc.SetArgs([]string{newHookName})
	Cc.Execute()
	project := projectfile.Get()

	var found = false
	for _, hook := range project.Hooks {
		if hook.Name == newHookName {
			found = true
		}
	}
	assert.False(t, found, fmt.Sprintf("Should NOT find a hook named %v", newHookName))
	err = removeTmpDir()
	assert.Nil(t, err, "Tried to remove tmp testing dir")
}

// Test it doesn't explode when run with no args
func TestExecute(t *testing.T) {
	Args.Hook, Args.Command = "", "" // reset
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	Command.Execute()

	assert.Equal(t, true, true, "Execute didn't panic")
}

//
func TestAddHookFailIdentical(t *testing.T) {
	Args.Hook, Args.Command = "", "" // reset
	project := projectfile.Get()
	err := moveToTmpDir()
	assert.Nil(t, err, "A temporary directory was created and entered as CWD")

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

	err = removeTmpDir()
	assert.Nil(t, err, "Tried to remove tmp testing dir")
}
