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
	tmpdir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	fileutils.Touch(filepath.Join(tmpdir, "state.exe"))

	assert.Equal(t, filepath.Join(tmpdir, "state.exe"), findExecutable("state", "/other_path;"+tmpdir, ".COM;.EXE;.BAT"))
	assert.Equal(t, filepath.Join(tmpdir, "state.EXE"), findExecutable("state.EXE", "/other_path;"+tmpdir, ".COM;.EXE;.BAT"))
	assert.Equal(t, filepath.Join(tmpdir, "state.exe"), findExecutable("state.exe", "/other_path;"+tmpdir, ".COM;.EXE;.BAT"))
	assert.Equal(t, "", findExecutable("state", "/other_path;"+tmpdir, ".COM;.EXE;.BAT"))
	assert.Equal(t, "", findExecutable("non-existent", "/other_path;"+tmpdir, ".COM;.EXE;.BAT"))
	assert.Equal(t, "", findExecutable("state", "/other_path", ".COM;.EXE;.BAT"))
}
