package export

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRecipe(t *testing.T, args []string) {
	root, err := environment.GetRootPath()
	require.NoError(t, err, "should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	cc := Command.GetCobraCmd()
	cc.SetArgs(append([]string{"recipe"}, args...))
}

func TestExportRecipe(t *testing.T) {
	setupRecipe(t, []string{"test"})

	err := Command.Execute()
	assert.NoError(t, err, "executed without error")
	assert.NoError(t, failures.Handled(), "no failure occurred")
}
