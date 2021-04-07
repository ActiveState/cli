package forwarder

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

func TestForwarder(t *testing.T) {
	fw, err := New(map[string]string{}, "/project/path")
	require.NoError(t, err, errs.Join(err, ": "))

	exePath := "/i/am/an/exe/"
	exes := []string{exePath + "a", exePath + "b", exePath + "c"}

	t.Run("Create forwarders", func(t *testing.T) {
		err = fw.updateWithExes(exes)
		require.NoError(t, err, errs.Join(err, ": "))
	})

	// Verify forwarders
	for _, exe := range exes {
		path := filepath.Join(fw.BinPath(), nameForwarder(filepath.Base(exe)))
		t.Run("Forwarder Exists", func(t *testing.T) {
			if !fileutils.FileExists(path) {
				t.Errorf("Could not locate exe: %s", path)
				t.FailNow()
			}
		})

		t.Run("Forwarder containts expected executable", func(t *testing.T) {
			contains, err := fileutils.FileContains(path, []byte(exe))
			require.NoError(t, err, errs.Join(err, ": "))
			if !contains {
				t.Errorf("File %s does not contain %q, contents: %q", path, exe, fileutils.ReadFileUnsafe(path))
				t.FailNow()
			}
		})
	}

	t.Run("Cleanup old forwarders", func(t *testing.T) {
		err = fw.Cleanup([]string{exes[1]})
		require.NoError(t, err, errs.Join(err, ": "))

		files := fileutils.ListDir(fw.BinPath(), false)
		require.Len(t, files, 1, "Cleanup should only keep one exe")
		require.Equal(t, filepath.Base(exes[1]), filepath.Base(files[0]), "Cleanup should leave the exe we requested")
	})

	t.Run("Update doesn't needlessly write", func(t *testing.T) {
		// Verify that another update doesn't needlessly write the same forwarder again
		files := fileutils.ListDir(fw.BinPath(), false)
		modtime, err := fileutils.ModTime(files[0])
		require.NoError(t, err, errs.Join(err, ": "))

		err = fw.updateWithExes([]string{exes[1]})
		require.NoError(t, err, errs.Join(err, ": "))

		newModtime, err := fileutils.ModTime(files[0])
		require.NoError(t, err, errs.Join(err, ": "))

		assert.Equal(t, modtime, newModtime, "Exe should not have been updated as the old value is still valid")
	})
}

func TestNameForwarder(t *testing.T) {
	if runtime.GOOS != "windows" {
		return // Pointless to test outside windows
	}

	assert.Equal(t, "filename.bat", nameForwarder("filename.exe"))
}
