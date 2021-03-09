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

	cases := []struct {
		name string
		keep string
	}{
		{"PID still running", "keep"},
		{"PID file deleted", "remove"},
	}

	for _, tc := range cases {
		t.Run("locked in other process with "+tc.name, func(tt *testing.T) {
			lockFile := filepath.Join(tmpDir, "locked-"+tc.keep)

			lockCmd := exec.Command(lockerExe, lockFile, tc.keep)
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

			pl, err := NewPidLock(lockFile)
			require.NoError(tt, err)

			// trying to acquire the lock in this process should fail
			ok, err := pl.TryLock()
			assert.False(tt, ok)
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

		t.Run("stress-test with "+tc.name, func(tt *testing.T) {
			// stress tests runs numProcesses in parallel, and only one should get the lock
			numProcesses := 10

			lockFile := filepath.Join(tmpDir, "stress-test-"+tc.keep)

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
					lockCmd := exec.Command(lockerExe, lockFile, tc.keep)
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
					<-ctx.Done()
					interruptProcess(tt, lockCmd.Process)

					err = lockCmd.Wait()
					require.NoError(tt, err)
					assert.True(tt, lockCmd.ProcessState.Exited())
					assert.Equal(tt, 0, lockCmd.ProcessState.ExitCode())
				}()
			}

			// timeout if test does not finish after 60 seconds
			go func() {
				select {
				case <-ctx.Done():
					return
				case <-time.After(60 * time.Second):
					done <- "TIMEOUT"
				}
			}()

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
}

func Test_acquirePidLock(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	lockFile := filepath.Join(tmpDir, "lockfile")

	pl, err := NewPidLock(lockFile)
	require.NoError(t, err)
	ok, err := pl.TryLock()
	assert.True(t, ok)
	require.NoError(t, err)

	pl2, err := NewPidLock(lockFile)
	require.NoError(t, err)
	ok2, err := pl2.TryLock()
	assert.False(t, ok2)
	assert.Error(t, err)

	err = pl2.Close()
	require.NoError(t, err)

	err = pl.Close()
	require.NoError(t, err)
	f, err := os.Stat(lockFile)
	assert.True(t, err != nil && f == nil)

	pl, err = NewPidLock(lockFile)
	require.NoError(t, err)
	ok, err = pl.TryLock()
	require.NoError(t, err)
	assert.True(t, ok)

	err = pl.Close(true)
	require.NoError(t, err)
	f, err = os.Stat(lockFile)
	assert.True(t, err == nil && !f.IsDir())

	pl, err = NewPidLock(lockFile)
	require.NoError(t, err)

	ok, err = pl.TryLock()
	assert.False(t, ok)
	assert.Error(t, err)

	err = pl.Close()
	require.NoError(t, err)
}

func TestPidExists(t *testing.T) {
	assert.True(t, PidExists(os.Getpid()))
	assert.False(t, PidExists(99999999))
}
