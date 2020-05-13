package sscommon

import (
	"os/exec"
	"syscall"

	"github.com/ActiveState/cli/internal/failures"
)

var lineBreak = "\n"
var lineBreakChar = `\n`

func stop(cmd *exec.Cmd) *failures.Failure {
	// may panic if process no longer exists
	defer failures.Recover()

	// darwin randomly returns an error when using cmd.Process.Signal
	sig := syscall.SIGHUP
	if err := syscall.Kill(cmd.Process.Pid, sig); err != nil {
		return FailSignalCmd.Wrap(err)
	}

	sig = syscall.SIGTERM
	if err := syscall.Kill(cmd.Process.Pid, sig); err != nil {
		return FailSignalCmd.Wrap(err)
	}

	return nil
}
