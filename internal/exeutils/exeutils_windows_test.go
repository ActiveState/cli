// +build windows

package exeutils

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_PathForExecutables(t *testing.T) {
	testDir := "C:\testdir"
	fileExists := func(fp string) bool {
		return strings.ToLower(fp) == strings.ToLower(filepath.Join(testDir, "state.exe"))
	}
	filter := func(string) bool { return true}

	assert.Equal(t, filepath.Join(testDir, "state.exe"), findExe("state", "/other_path;"+testDir, fileExists, filter))
	assert.Equal(t, filepath.Join(testDir, "state.EXE"), findExe("state.EXE", "/other_path;"+testDir, fileExists, filter))
	assert.Equal(t, filepath.Join(testDir, "state.exe"), findExe("state.exe", "/other_path;"+testDir, fileExists, filter))
	assert.Equal(t, "", findExe("state", "/other_path;"+testDir, fileExists, filter))
	assert.Equal(t, "", findExe("non-existent", "/other_path;"+testDir, fileExists, filter))
	assert.Equal(t, "", findExe("state", "/other_path", fileExists, filter))
}
