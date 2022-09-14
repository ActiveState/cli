package executor

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/errs"
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
	fw := NewInit(binPath)
	fw.setAltExecSrcPath(dummyExecSrc)

	exePath := "/i/am/an/exe/"
	exes := []string{exePath + "a", exePath + "b", exePath + "c"}
	env := map[string]string{"PATH": "exePath"}

	t.Run("Create executors", func(t *testing.T) {
		err = fw.Apply("/sock-path", target, env, exes)
		require.NoError(t, err, errs.Join(err, ": "))
	})

	// Verify executors
	for _, exe := range exes {
		path := filepath.Join(binPath, filepath.Base(exe))
		t.Run("Executor Exists", func(t *testing.T) {
			if !fileutils.FileExists(path) {
				t.Errorf("Could not locate exe: %s", path)
				t.FailNow()
			}
		})

		t.Run("Executor contains expected executable", func(t *testing.T) {
			contains, err := fileutils.FileContains(path, dummyExecData)
			require.NoError(t, err, errs.Join(err, ": "))
			if !contains {
				t.Errorf("File %s does not contain %q, contents: %q", path, exe, fileutils.ReadFileUnsafe(path))
				t.FailNow()
			}
		})
	}

	t.Run("Cleanup old executors", func(t *testing.T) {
		err = fw.Clean()
		require.NoError(t, err, errs.Join(err, ": "))

		files := fileutils.ListDirSimple(binPath, false)
		require.Len(t, files, 0, "Cleanup should remove all exes")
	})
}
