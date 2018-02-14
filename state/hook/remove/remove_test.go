package remove

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	helper "github.com/ActiveState/ActiveState-CLI/internal/helpers"
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

	hash, _ := helper.HashHookStruct(hook)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{hash})
	Command.Execute()
	mappedHooks, _ := helper.FilterHooks([]string{cmdName})
	assert.Equal(t, 0, len(mappedHooks), fmt.Sprintf("No hooks should be found of name: '%V'", cmdName))

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

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{cmdName})
	Command.Execute()
	mappedHooks, _ := helper.FilterHooks([]string{cmdName})
	assert.Equal(t, 0, len(mappedHooks), fmt.Sprintf("No hooks should be found of name: '%V'", cmdName))

	err = removeTmpDir()
	assert.Nil(t, err, "Tried to remove tmp testing dir")
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

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{cmdName})
	Command.Execute()
	mappedHooks, _ := helper.FilterHooks([]string{cmdName})
	assert.Equal(t, 2, len(mappedHooks[cmdName]), fmt.Sprintf("There should still be two commands of the same name in the config: '%v'", cmdName))

	err = removeTmpDir()
	assert.Nil(t, err, "Tried to remove tmp testing dir")
}
