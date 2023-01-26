package environment_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	_ "github.com/ActiveState/cli/internal-as/config"
	"github.com/ActiveState/cli/internal/environment"
)

func TestGetRootPath(t *testing.T) {
	rootPath, err := environment.GetRootPath()

	if err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(rootPath, "internal-as/environment/environment_test.go")
	require.FileExists(t, file)
}
