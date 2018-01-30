package environment

import (
	"testing"

	_ "github.com/ActiveState/ActiveState-CLI/internal/config"
	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/stretchr/testify/assert"
)

func TestGetRootPath(t *testing.T) {
	path, err := GetRootPath()

	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, path, constants.LibraryNamespace+constants.LibraryName, "Should detect root path")
}
