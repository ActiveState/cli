package sscommon

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/ActiveState/cli/internal/failures"
)

var (
	// FailExecCmd represents a failure running a cmd
	FailExecCmd = failures.Type("sscommon.fail.execcmd")

	// FailSignalCmd represents a failure sending a system signal to a cmd
	FailSignalCmd = failures.Type("sscommon.fail.signalcmd")
)

// Start wires stdin/stdout/stderr into the provided command, starts it, and
// returns a channel to monitor errors on.
func Start(cmd *exec.Cmd) chan *failures.Failure {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr

	cmd.Start()

	fs := make(chan *failures.Failure, 1)

	go func() {
		defer close(fs)

		if err := cmd.Wait(); err != nil {
			if eerr, ok := err.(*exec.ExitError); ok {
				code := eerr.ExitCode()
				if code == 130 || eerr.Exited() && code == -1 {
					return
				}
				fs <- FailExecCmd.Wrap(eerr)
				return
			}
		}
	}()

	return fs
}

// Stop signals the provided command to terminate.
func Stop(cmd *exec.Cmd) *failures.Failure {
	sig := syscall.SIGTERM
	if runtime.GOOS == "windows" {
		sig = syscall.SIGKILL
	}

	// may panic if process no longer exists
	defer failures.Recover()
	if err := cmd.Process.Signal(sig); err != nil {
		return FailSignalCmd.Wrap(err)
	}

	return nil
}
