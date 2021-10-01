// +build !windows

package exeutils

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_PathForExecutables(t *testing.T) {
	testDir := "/testDir"
	fileExists := func(f string) bool {
		return f == filepath.Join(testDir, "state")
	}

	assert.Equal(t, filepath.Join(testDir, "state"), findExecutables("state", "/other_path:"+testDir, fileExists))
	assert.Equal(t, "", findExecutables("non-existent", "/other_path:"+testDir, fileExists))
	assert.Equal(t, "", findExecutables("state", "/other_path", fileExists))
}
