package legacyupd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func setup(t *testing.T, withVersion bool) {
	cwd, err := environment.GetRootPath()
	require.NoError(t, err, "Should fetch cwd")
	path := filepath.Join(cwd, "internal", "updater", "testdata")
	if withVersion {
		path = filepath.Join(path, "withversion")
	}
	err = os.Chdir(path)
	require.NoError(t, err, "Should change dir without issue.")
	projectfile.Reset()
}
