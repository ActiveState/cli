package activate

import (
	"os"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/config"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/stretchr/testify/assert"
)

// Little hack to ensure we are ran before the init function in activate.go
var _ = func() (_ struct{}) {
	config.Init()
	locale.Init()
	return
}()

func TestMain(m *testing.M) {
	config.Init()
	locale.Init()
	code := m.Run()
	os.Exit(code)
}

var cloneURL = "https://github.com/golang/playground"

func TestExecute(t *testing.T) {
	assert := assert.New(t)

	path, cd = "", false // clear command line options
	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"activate"})
	Command.Execute()
	assert.Equal(true, true, "Execute didn't panic")
}

func TestExecuteURL(t *testing.T) {
	assert := assert.New(t)

	path, cd = "", false // clear command line options
	origCWD, _ := os.Getwd()
	Command.GetCobraCmd().SetArgs([]string{cloneURL})
	Command.Execute()
	assert.DirExists("playground", "git cloned into directory 'playground'")
	currentCWD, _ := os.Getwd()
	assert.Equal(origCWD, currentCWD, "The current directory did not change")
	os.RemoveAll("playground")
}

func TestExecuteURLPath(t *testing.T) {
	assert := assert.New(t)

	path, cd = "", false // clear command line options
	origCWD, _ := os.Getwd()
	Command.GetCobraCmd().SetArgs([]string{cloneURL, "--path", "play"})
	Command.Execute()
	assert.DirExists("play", "git cloned into directory 'play'")
	currentCWD, _ := os.Getwd()
	assert.Equal(origCWD, currentCWD, "The current directory did not change")
	os.RemoveAll("play")

	path, cd = "", false // clear command line options
	Command.GetCobraCmd().SetArgs([]string{"--path", "play", cloneURL})
	Command.Execute()
	assert.DirExists("play", "git cloned into directory 'play'")
	currentCWD, _ = os.Getwd()
	assert.Equal(origCWD, currentCWD, "The current directory did not change")
	os.RemoveAll("play")
}

func TestExecuteCD(t *testing.T) {
	assert := assert.New(t)

	path, cd = "", false // clear command line options
	origCWD, _ := os.Getwd()
	Command.GetCobraCmd().SetArgs([]string{cloneURL, "--cd"})
	Command.Execute()
	currentCWD, _ := os.Getwd()
	assert.NotEqual(origCWD, currentCWD, "The current directory changed")
	os.Chdir("..")
	assert.DirExists("playground", "git cloned into directory 'playground'")
	os.RemoveAll("playground")
}
