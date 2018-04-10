package hook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
	assert := assert.New(t)

	Command.Execute()

	assert.Equal(true, true, "Execute didn't panic")
}

func TestExecuteFiltered(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--filter", "FIRST_INSTALL"})

	Command.Execute()

	assert.Equal(true, true, "Execute didn't panic")
}
