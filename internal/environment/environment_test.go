package environment

import (
	"path/filepath"
	"testing"

	_ "github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/stretchr/testify/assert"
)

func TestTargetEnvironment(t *testing.T) {
	env := TargetEnvironment()
	assert.Contains(t, []Environment{Production, Development}, env, "Returns a valid environment")
}

func TestGetRootPath(t *testing.T) {
	rootPath, err := GetRootPath()

	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, rootPath, filepath.FromSlash(constants.LibraryNamespace+constants.LibraryName), "Should detect root path")
}
