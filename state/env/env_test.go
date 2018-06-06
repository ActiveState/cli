package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
)

// This is mostly a clone of the state/hooks/hook_test.go file. Any tests added,
// modified, or removed in that file should be applied here and vice-versa.

func TestExecute(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))
	assert := assert.New(t)

	Command.Execute()

	assert.Equal(true, true, "Execute didn't panic")
}
