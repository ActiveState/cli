// +build windows

package exeutils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PathForExecutables(t *testing.T) {
	testDir := "C:\testdir"
	fileExists := func(fp string) bool {
		return strings.ToLower(fp) == strings.ToLower(filepath.Join(testDir, "state.exe"))
	}

	assert.Equal(t, filepath.Join(testDir, "state.exe"), findExecutable("state", "/other_path;"+tmpdir, ".COM;.EXE;.BAT", fileExists))
	assert.Equal(t, filepath.Join(testDir, "state.EXE"), findExecutable("state.EXE", "/other_path;"+tmpdir, ".COM;.EXE;.BAT", fileExists))
	assert.Equal(t, filepath.Join(testDir, "state.exe"), findExecutable("state.exe", "/other_path;"+tmpdir, ".COM;.EXE;.BAT", fileExists))
	assert.Equal(t, "", findExecutable("state", "/other_path;"+tmpdir, "", fileExists))
	assert.Equal(t, "", findExecutable("non-existent", "/other_path;"+tmpdir, ".COM;.EXE;.BAT", fileExists))
	assert.Equal(t, "", findExecutable("state", "/other_path", ".COM;.EXE;.BAT", fileExists))
}
