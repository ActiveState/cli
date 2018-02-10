package add

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"

	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

// Test it doesn't explode when run with no args
func TestExecute(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	Command.Execute()

	assert.Equal(t, true, true, "Execute didn't panic")
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

//  This test MUST go before TestAddHookPass.  Something to do with writing files.
func TestAddHookFail(t *testing.T) {
	root, err := environment.GetRootPath()
	testDir := filepath.Join(root, "test")
	os.Chdir(testDir)
	assert.NoError(t, err, "Should detect root path")

	config, _ := projectfile.Get()
	Cc := Command.GetCobraCmd()
	newHookName := "A_HOOK"
	Cc.SetArgs([]string{newHookName})
	Cc.Execute()

	var found = false
	for _, hook := range config.Hooks {
		if hook.Name == newHookName {
			found = true
		}
	}
	assert.False(t, found, fmt.Sprintf("Should NOT find a hook named %v", newHookName))
}
func TestAddHookPass(t *testing.T) {
	root, err := environment.GetRootPath()
	testDir := filepath.Join(root, "test")
	savedconfigPath := filepath.Join(testDir, constants.ConfigFileName+".orig")
	configPath := filepath.Join(testDir, constants.ConfigFileName)
	os.Chdir(testDir)
	_ = copy(configPath, savedconfigPath)
	assert.NoError(t, err, "Should detect root path")

	config, _ := projectfile.Get()
	newHookName := "A_HOOK"
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{newHookName, "echo 'This is a command'"})
	Cc.Execute()

	var found = false
	for _, hook := range config.Hooks {
		if hook.Name == newHookName {
			found = true
		}
	}
	assert.True(t, found, fmt.Sprintf("Should find a hook named %v", newHookName))

	os.Remove(configPath)
	_ = copy(savedconfigPath, configPath)
	os.Remove(savedconfigPath)
}
