package remove

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	helper "github.com/ActiveState/ActiveState-CLI/internal/helpers"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestExecute(t *testing.T) {
	assert := assert.New(t)
	Command.Execute()
	assert.Equal(true, true, "Execute didn't panic")
}

func TestRemoveByHash(t *testing.T) {
	root, _ := environment.GetRootPath()
	os.Chdir(filepath.Join(root, "test"))
	config, _ := projectfile.Get()
	cmdName := "REMOVE_ME"

	hook := projectfile.Hook{Name: cmdName, Value: "Weeeee, I'm command!  I do you a heckin command! Teehee!"}
	config.Hooks = append(config.Hooks, hook)

	hash, _ := helper.HashHookStruct(hook)
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{hash})
	Command.Execute()
	mappedHooks, _ := helper.FilterHooks([]string{cmdName})
	assert.Equal(t, 0, len(mappedHooks), fmt.Sprintf("No hooks should be found of name: '%V'", cmdName))
}

func TestRemoveByName(t *testing.T) {
	root, _ := environment.GetRootPath()
	os.Chdir(filepath.Join(root, "test"))
	config, _ := projectfile.Get()
	cmdName := "REMOVE_ME"

	hook := projectfile.Hook{Name: cmdName, Value: "Weeeee, I'm command!  I do you a heckin command! Teehee!"}
	config.Hooks = append(config.Hooks, hook)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{cmdName})
	Command.Execute()
	mappedHooks, _ := helper.FilterHooks([]string{cmdName})
	assert.Equal(t, 0, len(mappedHooks), fmt.Sprintf("No hooks should be found of name: '%V'", cmdName))
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
	savedconfigPath := filepath.Join(testDir, constants.ConfigFileName+".orig")
	configPath := filepath.Join(testDir, constants.ConfigFileName)
	os.Chdir(testDir)
	_ = copy(configPath, savedconfigPath)
	assert.NoError(t, err, "Should detect root path")
	cmdName := "REMOVE_ME"
	config, _ := projectfile.Get()

	hook1 := projectfile.Hook{Name: cmdName, Value: "Weeeee, I'm command!  I do you a heckin command! Teehee!"}
	hook2 := projectfile.Hook{Name: cmdName, Value: "Weeeee, I'm command!  I do you a heckin command! Teehee!!"}
	config.Hooks = append(config.Hooks, hook1)
	config.Hooks = append(config.Hooks, hook2)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{cmdName})
	Command.Execute()
	mappedHooks, _ := helper.FilterHooks([]string{cmdName})
	assert.Equal(t, 2, len(mappedHooks[cmdName]), fmt.Sprintf("There should still be two commands of the same name in the config: '%v'", cmdName))

	os.Remove(configPath)
	_ = copy(savedconfigPath, configPath)
	os.Remove(savedconfigPath)
}
