package executors

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
)

func TestExecutor(t *testing.T) {
	binPath, err := ioutil.TempDir("", "executor")
	require.NoError(t, err, errs.Join(err, ": "))

	dummyExecData := []byte("junk state-exec junk")
	dummyExecSrc := "binPath/SRC"
	err = fileutils.WriteFile(dummyExecSrc, dummyExecData)
	defer func() { _ = os.RemoveAll(filepath.Dir(dummyExecSrc)) }()
	require.NoError(t, err, errs.Join(err, ": "))

	target := target.NewCustomTarget("owner", "project", "1234abcd-1234-abcd-1234-abcd1234abcd", "dummy/path", target.NewExecTrigger("test"), false)
	execInit := New(binPath)
	execInit.altExecSrcPath = dummyExecSrc

	exePath := "/i/am/an/exe/"
	exes := []string{exePath + "a", exePath + "b", exePath + "c"}
	winExes := []string{exePath + "d" + exeutils.Extension, exePath + "e" + exeutils.Extension}
	allExes := exes
	if runtime.GOOS == "windows" {
		allExes = append(allExes, winExes...)
	}
	env := map[string]string{"PATH": "exePath"}

	t.Run("Create executors", func(t *testing.T) {
		err = execInit.Apply("/sock-path", target, env, exes)
		require.NoError(t, err, errs.Join(err, ": "))
	})

	// Verify executors
	for i, exe := range allExes {
		path := filepath.Join(binPath, filepath.Base(exe))

		if runtime.GOOS == "windows" && i < len(exes) { // ensure exes are not represented
			t.Run("Executor Exists", func(t *testing.T) {
				if fileutils.FileExists(path) {
					t.Errorf("Should not locate exe: %s", path)
				}
			})
			continue
		}

		t.Run("Executor Exists", func(t *testing.T) {
			if !fileutils.FileExists(path) {
				t.Errorf("Could not locate exe: %s", path)
			}
		})

		t.Run("Executor contains expected executable", func(t *testing.T) {
			contains, err := fileutils.FileContains(path, dummyExecData)
			require.NoError(t, err, errs.Join(err, ": "))
			if !contains {
				t.Errorf("File %s does not contain %q, contents: %q", path, exe, fileutils.ReadFileUnsafe(path))
			}
		})
	}

	// add legacy files - deprecated
	require.NoError(t, fileutils.WriteFile(path.Join(binPath, "old_exec"), []byte(legacyExecutorDenoter)))
	require.NoError(t, fileutils.WriteFile(path.Join(binPath, "old_shim"), []byte(legacyShimDenoter)))

	t.Run("Cleanup old executors", func(t *testing.T) {
		err = execInit.Clean()
		require.NoError(t, err, errs.Join(err, ": "))

		files := fileutils.ListDirSimple(binPath, false)
		require.Len(t, files, 0, "Cleanup should remove all exes")
	})
}
