package executor

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

func TestExecutor(t *testing.T) {
	fw, err := New("/project/path")
	require.NoError(t, err, errs.Join(err, ": "))

	exePath := "/i/am/an/exe/"
	exes := []string{exePath + "a", exePath + "b", exePath + "c"}
	env := map[string]string{"PATH": "exePath"}

	t.Run("Create executors", func(t *testing.T) {
		err = fw.Update("dummy/project", "dummy/sock", env, exes)
		require.NoError(t, err, errs.Join(err, ": "))
	})

	// Verify executors
	for _, exe := range exes {
		path := filepath.Join(fw.BinPath(), NameForExe(filepath.Base(exe)))
		t.Run("Executor Exists", func(t *testing.T) {
			if !fileutils.FileExists(path) {
				t.Errorf("Could not locate exe: %s", path)
				t.FailNow()
			}
		})

		t.Run("Executor containts expected executable", func(t *testing.T) {
			contains, err := fileutils.FileContains(path, []byte(exe))
			require.NoError(t, err, errs.Join(err, ": "))
			if !contains {
				t.Errorf("File %s does not contain %q, contents: %q", path, exe, fileutils.ReadFileUnsafe(path))
				t.FailNow()
			}
		})
	}

	t.Run("Cleanup old executors", func(t *testing.T) {
		err = fw.Cleanup()
		require.NoError(t, err, errs.Join(err, ": "))

		files := fileutils.ListDirSimple(fw.BinPath(), false)
		require.Len(t, files, 0, "Cleanup should remove all exes")
	})
}

func TestNameForExe(t *testing.T) {
	if runtime.GOOS != "windows" {
		return // Pointless to test outside windows
	}

	assert.Equal(t, "filename.bat", NameForExe("filename.exe"))
}
