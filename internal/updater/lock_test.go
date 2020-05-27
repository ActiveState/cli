package updater

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
		filepath.Join(root, "internal", "updater", "testdata", "locker"),
	)
	err = cmd.Run()
	require.NoError(t, err)

	return lockerExe
}

func Test_acquireUpdateLockProcesses(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// build the test process that acquires a lock
	lockerExe := buildTestExecutable(t, tmpDir)

	t.Run("locked in other process", func(tt *testing.T) {
		updateDir := filepath.Join(tmpDir, "locked")
		err = os.MkdirAll(updateDir, 0755)
		require.NoError(tt, err)

		lockCmd := exec.Command(lockerExe, updateDir)
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
		ok, cleanup := AcquireUpdateLock(updateDir)
		assert.False(tt, ok)
		assert.Nil(tt, cleanup)

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

		updateDir := filepath.Join(tmpDir, "stress")
		err = os.MkdirAll(updateDir, 0755)
		require.NoError(tt, err)

		done := make(chan string, numProcesses+1)
		defer close(done)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for i := 0; i < numProcesses; i++ {
			go func() {
				var s string = "BLOCKED"
				defer func() { done <- s }()
				lockCmd := exec.Command(lockerExe, updateDir)
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

func Test_acquireUpdateLock(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	ok, cleanup := AcquireUpdateLock(tmpDir)
	assert.True(t, ok)
	assert.NotNil(t, cleanup)

	ok2, cleanup2 := AcquireUpdateLock(tmpDir)
	assert.False(t, ok2)
	assert.Nil(t, cleanup2)

	cleanup()
	ok, cleanup = AcquireUpdateLock(tmpDir)
	assert.True(t, ok)
	assert.NotNil(t, cleanup)

	cleanup()
}
