package osutils

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
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
		lockFile := filepath.Join(tmpDir, "locked")

		lockCmd := exec.Command(lockerExe, lockFile)
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

		// trying to acquire the lock in this process should fail
		pl, err := NewPidLock(lockFile)
		assert.Nil(tt, pl)
		assert.Error(tt, err)

		// stopping the other process
		err = lockCmd.Process.Signal(os.Interrupt)
		require.NoError(tt, err)

		// waiting for the process to finish without error
		err = lockCmd.Wait()
		require.NoError(tt, err)
		assert.True(tt, lockCmd.ProcessState.Exited())
		assert.Equal(tt, 0, lockCmd.ProcessState.ExitCode())
	})

	t.Run("stress-test", func(tt *testing.T) {
		// stress tests runs numProcesses in parallel, and only one should get the lock
		numProcesses := 100

		lockFile := filepath.Join(tmpDir, "stress")

		done := make(chan string, numProcesses+1)
		defer close(done)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for i := 0; i < numProcesses; i++ {
			go func() {
				var s string = "BLOCKED"
				defer func() { done <- s }()
				lockCmd := exec.Command(lockerExe, lockFile)
				stdout, err := lockCmd.StdoutPipe()
				require.NoError(tt, err)
				err = lockCmd.Start()
				require.NoError(tt, err)

				// wait for command to block
				buf := make([]byte, 6)
				n, err := stdout.Read(buf)
				require.NoError(tt, err)
				require.Equal(tt, 6, n)
				if string(buf) == "DENIED" {
					s = "DENIED"
					return
				}

				// if we get here, the process acquired the lock
				assert.Equal(tt, "LOCKED", string(buf))

				// wait for the signal to kill process and to release the lock
				<-ctx.Done()
				err = lockCmd.Process.Signal(os.Interrupt)
				require.NoError(tt, err)

				err = lockCmd.Wait()
				require.NoError(tt, err)
				assert.True(tt, lockCmd.ProcessState.Exited())
				assert.Equal(tt, 0, lockCmd.ProcessState.ExitCode())
			}()
		}

		// timeout if test does not finish after five seconds
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				done <- "TIMEOUT"
			}
		}()

		// ensure that numProcesses-1 processes are denied access and only 1 got the lock
		var count int
		for d := range done {
			if d == "TIMEOUT" {
				tt.Fatalf("test timed out")
			}
			count++
			if count <= numProcesses-1 {
				assert.Equal(tt, "DENIED", d)
			}
			if count == numProcesses-1 {
				cancel()
			}
			if count == numProcesses {
				assert.Equal(tt, "BLOCKED", d)
				break
			}
		}
	})
}

func Test_acquirePidLock(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	lockFile := filepath.Join(tmpDir, "lockfile")

	pl, err := NewPidLock(lockFile)
	assert.NotNil(t, pl)
	assert.NoError(t, err)

	pl2, err := NewPidLock(lockFile)
	assert.Nil(t, pl2)
	assert.Error(t, err)

	pl.Close()

	pl, err = NewPidLock(lockFile)
	assert.NotNil(t, pl)
	assert.NoError(t, err)
	pl.Close()
}
