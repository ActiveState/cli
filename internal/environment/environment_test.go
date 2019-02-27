package environment_test

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"

	_ "github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/stretchr/testify/assert"
)

func TestGetRootPath(t *testing.T) {
	rootPath, err := environment.GetRootPath()

	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, rootPath, filepath.FromSlash(constants.LibraryNamespace+constants.LibraryName), "Should detect root path")
}
