package sscommon

import (
	"os/exec"
	"syscall"

	"github.com/ActiveState/cli/internal/failures"
)

func stop(cmd *exec.Cmd) *failures.Failure {
	sig := syscall.SIGTERM

	// may panic if process no longer exists
	defer failures.Recover()
	// darwin randomly returns an error when using cmd.Process.Signal
	if err := syscall.Kill(cmd.Process.Pid, sig); err != nil {
		return FailSignalCmd.Wrap(err)
	}

	return nil
}
