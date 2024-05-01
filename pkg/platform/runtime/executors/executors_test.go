package executors

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
)

func TestExecutor(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "as-executor-test")
	require.NoError(t, err, errs.JoinMessage(err))
	defer func() { _ = os.RemoveAll(tmpDir) }()

	dummyExecData := []byte("junk state-exec junk")
	dummyExecSrc := filepath.Join(tmpDir, "_SRC")

	err = fileutils.WriteFile(dummyExecSrc, dummyExecData)
	require.NoError(t, err, errs.JoinMessage(err))

	target := target.NewCustomTarget("owner", "project", "1234abcd-1234-abcd-1234-abcd1234abcd", "dummy/path", target.NewExecTrigger("test"))
	execDir := filepath.Join(tmpDir, "exec")
	execInit := New(execDir)
	execInit.altExecSrcPath = dummyExecSrc

	exec := func(in string) string { return filepath.Join(tmpDir, in) }
	exes := make(map[string]string) // map[string]string{ "executable": "executor" }
	switch runtime.GOOS {
	case "windows":
		exes["a.exe"] = "exec/a.exe"
		exes["b.bat"] = "exec/b.exe"
		exes["c.cmd"] = "exec/c.exe"

	default:
		exes["bin/a"] = "exec/a"
		exes["bin/b.sh"] = "exec/b.sh"
	}
	env := map[string]string{"PATH": execDir}
	var inputExes []string
	for exe := range exes {
		inputExes = append(inputExes, exe)
	}

	t.Run("Create executors", func(t *testing.T) {
		err = execInit.Apply("/sock-path", target, env, inputExes)
		require.NoError(t, err, errs.JoinMessage(err))
	})

	// Verify executors
	for _, utor := range exes {
		executor := exec(utor)

		t.Run("Executor Exists", func(t *testing.T) {
			if !fileutils.FileExists(executor) {
				t.Errorf("Could not locate executor: %s", executor)
			}
		})

		t.Run("Executor contains expected executable", func(t *testing.T) {
			contains, err := fileutils.FileContains(executor, dummyExecData)
			require.NoError(t, err, errs.JoinMessage(err))
			if !contains {
				t.Errorf("File %s does not contain %q, contents: %q", executor, dummyExecData, fileutils.ReadFileUnsafe(executor))
			}
		})
	}

	// add legacy files - deprecated
	require.NoError(t, fileutils.WriteFile(exec("exec/old_exec"), []byte(legacyExecutorDenoter)))
	require.NoError(t, fileutils.WriteFile(exec("exec/old_shim"), []byte(legacyShimDenoter)))

	t.Run("Cleanup old executors", func(t *testing.T) {
		err = execInit.Clean()
		require.NoError(t, err, errs.JoinMessage(err))

		files, err := fileutils.ListDirSimple(exec("exec"), false)
		require.NoError(t, err, errs.JoinMessage(err))
		require.Len(t, files, 0, "Cleanup should remove all exes")
	})
}
