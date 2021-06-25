package lockfile

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"io/ioutil"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildTestExecutable(t *testing.T, dir string) string {
	root, err := environment.GetRootPath()
	require.NoError(t, err)
	lockerExe := filepath.Join(dir, "locker")
	if runtime.GOOS == "windows" {
		lockerExe += ".exe"
	}

	cmd := exec.Command(
		"go", "build", "-o", lockerExe,
		filepath.Join(root, "internal", "osutils", "testdata", "locker"),
	)
	err = cmd.Run()
	require.NoError(t, err)

	return lockerExe
}

func Test_acquirePidLockProcesses(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// build the test process that acquires a lock
	lockerExe := buildTestExecutable(t, tmpDir)

	t.Run("locked in other process", func(tt *testing.T) {
		lockFile := filepath.Join(tmpDir, "locked-")

		lockCmd := exec.Command(lockerExe, lockFile)
		lockCmd = prepLockCmd(lockCmd)

		stdout, err := lockCmd.StdoutPipe()
		require.NoError(tt, err)
		err = lockCmd.Start()
		require.NoError(tt, err)

		// wait for command to block
		buf := make([]byte, 6)
		n, err := stdout.Read(buf)
		require.NoError(tt, err)
		require.Equal(tt, 6, n)
		assert.Equal(tt, "LOCKED", string(buf))

		pl, err := NewLock(lockFile)
		require.NoError(tt, err)

		// trying to acquire the lock in this process should fail
		err = pl.TryLock()
		require.Error(tt, err)
		alreadyErr := &AlreadyLockedError{}
		assert.True(tt, errors.As(err, &alreadyErr))

		err = pl.Close()
		require.NoError(tt, err)

		// stopping the other process
		interruptProcess(tt, lockCmd.Process)

		// waiting for the process to finish without error
		err = lockCmd.Wait()
		require.NoError(tt, err)
		assert.True(tt, lockCmd.ProcessState.Exited())
		assert.Equal(tt, 0, lockCmd.ProcessState.ExitCode())
	})

	t.Run("stress-test", func(tt *testing.T) {
		// stress tests runs numProcesses in parallel, and only one should get the lock
		numProcesses := 10

		lockFile := filepath.Join(tmpDir, "stress-test")

		done := make(chan string, numProcesses+1)
		defer close(done)
		var wg sync.WaitGroup
		defer wg.Wait()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for i := 0; i < numProcesses; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				lockCmd := exec.Command(lockerExe, lockFile)
				lockCmd = prepLockCmd(lockCmd)
				stdout, err := lockCmd.StdoutPipe()
				require.NoError(tt, err)
				err = lockCmd.Start()
				require.NoError(tt, err)

				// wait for command to block
				buf := make([]byte, 6)
				n, err := stdout.Read(buf)
				require.NoError(tt, err)
				require.Equal(tt, 6, n)

				sb := string(buf[:n])
				if sb == "DENIED" {
					done <- "DENIED"
					return
				}
				assert.Equal(tt, "LOCKED", string(buf[:6]))

				done <- "LOCKED"

				// wait for the signal to kill process and to release the lock
				select {
				case <-ctx.Done():
					return
				case <-time.After(60 * time.Second):
					done <- "TIMEOUT"
				}
				interruptProcess(tt, lockCmd.Process)

				err = lockCmd.Wait()
				require.NoError(tt, err)
				assert.True(tt, lockCmd.ProcessState.Exited())
				assert.Equal(tt, 0, lockCmd.ProcessState.ExitCode())
			}()
		}

		// ensure that numProcesses-1 processes are denied access and only 1 got the lock
		var denied int
		var locked int
		for d := range done {
			if d == "TIMEOUT" {
				tt.Fatalf("test timed out")
			}
			if d == "LOCKED" {
				locked++
			}
			if d == "DENIED" {
				denied++
			}
			if denied+locked == numProcesses {
				break
			}
		}
		assert.Equal(t, 1, locked, "only process should lock")
		assert.Equal(t, numProcesses-1, denied, "all but on processes should have been denied lock-file access")
	})
}

func Test_acquirePidLock(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	lockFile := filepath.Join(tmpDir, "lockfile")

	pl, err := NewLock(lockFile)
	require.NoError(t, err)
	err = pl.TryLock()
	require.NoError(t, err)

	// This demonstrates that two locks in the same process are allowed.  You'll need to use other mechanisms to achieve synchronization inside the process.
	pl2, err := NewLock(lockFile)
	require.NoError(t, err, "should pidlock on existing file with existing lock")
	err = pl2.TryLock()
	if runtime.GOOS == "windows" {
		assert.Error(t, err, "on Windows a process can lock a file only once")
	} else {
		assert.NoError(t, err, "same process should be able to lock file again")
	}

	err = pl2.Close()
	require.NoError(t, err, "should close the second lock successfully")

	err = pl.Close()
	require.NoError(t, err, "should be able to close")
	assert.FileExists(t, lockFile)
}
