package sscommon

import (
	"os/exec"
	"syscall"

	"github.com/ActiveState/cli/internal/failures"
)

func stop(cmd *exec.Cmd) *failures.Failure {
	sig = syscall.SIGKILL

	// may panic if process no longer exists
	defer failures.Recover()
	if err := cmd.Process.Signal(sig); err != nil {
		return FailSignalCmd.Wrap(err)
	}

	return nil
}
