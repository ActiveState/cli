package environment_test

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"

	_ "github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRootPath(t *testing.T) {
	rootPath, err := environment.GetRootPath()

	if err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(rootPath, "activestate.yaml")
	require.FileExists(t, file)
	assert.Contains(t, string(fileutils.ReadFileUnsafe(file))[0:50], "name: cli")
}
