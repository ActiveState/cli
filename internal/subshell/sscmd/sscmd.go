package sscmd

import (
	"os/exec"
	"syscall"

	"github.com/ActiveState/cli/internal/failures"
)

var (
	// FailExecCmd represents a failure running a cmd
	FailExecCmd = failures.Type("sscmd.fail.execcmd")

	// FailSignalCmd represents a failure sending a system signal to a cmd
	FailSignalCmd = failures.Type("sscmd.fail.signalcmd")
)

// Start starts the provided command and returns a channel to monitor errors on.
func Start(cmd *exec.Cmd) chan *failures.Failure {
	cmd.Start()

	fs := make(chan *failures.Failure, 1)

	go func() {
		defer close(fs)

		if err := cmd.Wait(); err != nil {
			if eerr, ok := err.(*exec.ExitError); ok {
				if eerr.Exited() && eerr.ExitCode() == -1 {
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
	// may panic if process no longer exists
	defer failures.Recover()
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return FailSignalCmd.Wrap(err)
	}

	return nil
}
